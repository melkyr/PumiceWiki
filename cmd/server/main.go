package main

import (
	"context"
	"errors"
	"fmt"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/handler"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/cache"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/casbin/casbin/v2"
)

func main() {
	// --- Configuration Loading ---
	cfg, err := config.LoadConfig()
	if err != nil {
		// Use fmt.Printf here because the logger is not yet initialized.
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// --- Logger Initialization ---
	log := logger.New(cfg.Log)

	// --- Pre-flight Checks ---
	if cfg.Session.SecretKey == "" || cfg.Session.SecretKey == "CHANGE_ME_IN_PRODUCTION_SECRET!!" {
		log.Fatal(errors.New("session secret key not set"), "Please set a secure WIKI_SESSION_SECRETKEY environment variable.")
	}

	// --- Database Initialization and Migration ---
	log.Info("Applying database migrations...")
	if err := data.ApplyMigrations(cfg.DB.DSN, "migrations"); err != nil {
		log.Fatal(err, "Failed to apply migrations")
	}
	log.Info("Migrations applied successfully.")

	log.Info("Connecting to the database...")
	db, err := data.NewDB(cfg.DB)
	if err != nil {
		log.Fatal(err, "Failed to connect to database")
	}
	defer db.Close()
	log.Info("Database connection successful.")

	// --- Session Management Setup ---
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db.DB)
	sessionManager.Lifetime = time.Duration(cfg.Session.Lifetime) * time.Hour
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = cfg.Server.TLS.Enabled

	// --- Authentication and Authorization Setup ---
	log.Info("Initializing authentication and authorization...")
	authenticator, err := auth.NewAuthenticator(&cfg.OIDC)
	if err != nil {
		log.Fatal(err, "Failed to initialize authenticator")
	}
	enforcer, err := auth.NewEnforcer("mysql", cfg.DB.DSN, "auth_model.conf")
	if err != nil {
		log.Fatal(err, "Failed to initialize enforcer")
	}
	seedDefaultPolicies(enforcer, log)
	log.Info("Auth components initialized and policies seeded.")

	// --- View Template Initialization ---
	log.Info("Initializing view templates...")
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		log.Fatal(err, "Failed to initialize view templates")
	}
	log.Info("View templates initialized.")

	// --- Cache Initialization ---
	log.Info("Initializing SQLite cache...")
	cache, err := cache.New(cfg.Cache)
	if err != nil {
		log.Fatal(err, "Failed to initialize cache")
	}
	defer cache.Close()
	log.Info("Cache initialized.")

	// --- Dependency Injection and Handler Initialization ---
	// Initialize the application layers, injecting dependencies from top to bottom.
	pageRepository := data.NewSQLPageRepository(db)
	categoryRepository := data.NewCategoryRepository(db)
	pageService := service.NewPageService(pageRepository, categoryRepository, cache)
	pageHandler := handler.NewPageHandler(pageService, viewService, log)
	authHandler := handler.NewAuthHandler(authenticator, sessionManager, enforcer)
	seoHandler := handler.NewSeoHandler(pageService)

	authzMiddleware := middleware.Authorizer(enforcer, sessionManager)
	errorMiddleware := middleware.Error(log, viewService)

	// --- Router Setup ---
	// The router is the central hub that directs incoming requests to the correct handlers.
	router := handler.NewRouter(pageHandler, authHandler, seoHandler, authzMiddleware, errorMiddleware, sessionManager)

	// --- Server Initialization and Graceful Shutdown ---
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}
	go func() {
		if cfg.Server.TLS.Enabled {
			log.Info(fmt.Sprintf("Starting HTTPS server on %s", server.Addr))
			if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err, "Could not start HTTPS server")
			}
		} else {
			log.Info(fmt.Sprintf("Starting HTTP server on %s", server.Addr))
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err, "Could not start HTTP server")
			}
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Warn("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err, "Server forced to shutdown")
	}
	log.Info("Server exiting")
}

// seedDefaultPolicies ensures that the application has a baseline set of authorization rules.
// It checks if each default policy exists before adding it, making the operation idempotent
// and safe to run on every application start.
func seedDefaultPolicies(e casbin.IEnforcer, log logger.Logger) {
	log.Info("Seeding default authorization policies...")

	// Default policies grant basic access to anonymous users and content management
	// permissions to editors. Note that the 'editor' role inherits from 'anonymous'.
	policies := [][]string{
		// Anonymous users can view pages and access login/callback routes.
		{"anonymous", "/view/*", "GET"},
		{"anonymous", "/auth/login", "GET"},
		{"anonymous", "/auth/callback", "GET"},

		// Editors can do everything anonymous users can, plus edit, save, and list pages.
		{"editor", "/edit/*", "GET"},
		{"editor", "/save/*", "POST"},
		{"editor", "/list", "GET"},
	}
	for _, p := range policies {
		if has, _ := e.HasPolicy(p); !has {
			if _, err := e.AddPolicy(p); err != nil {
				log.Error(err, fmt.Sprintf("Failed to add policy %v", p))
			}
		}
	}

	// Granting the 'editor' role all permissions of the 'anonymous' role.
	if has, _ := e.HasRoleForUser("editor", "anonymous"); !has {
		if _, err := e.AddRoleForUser("editor", "anonymous"); err != nil {
			log.Error(err, "Failed to add role 'editor' -> 'anonymous'")
		}
	}
	log.Info("Policy seeding complete.")
}
