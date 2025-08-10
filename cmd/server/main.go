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
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log)

	if cfg.Session.SecretKey == "" || cfg.Session.SecretKey == "CHANGE_ME_IN_PRODUCTION_SECRET!!" {
		log.Fatal(errors.New("session secret key not set"), "Please set a secure WIKI_SESSION_SECRETKEY environment variable.")
	}

	log.Info("Applying database migrations...")
	if err := data.ApplyMigrations(cfg.DB.DSN, "/migrations"); err != nil {
		log.Fatal(err, "Failed to apply migrations")
	}
	log.Info("Migrations applied successfully.")

	log.Info("Connecting to the database...")
	db, err := data.NewDB(cfg.DB.DSN)
	if err != nil {
		log.Fatal(err, "Failed to connect to database")
	}
	defer db.Close()
	log.Info("Database connection successful.")

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db.DB)
	sessionManager.Lifetime = time.Duration(cfg.Session.Lifetime) * time.Hour
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = cfg.Server.TLS.Enabled

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

	log.Info("Initializing view templates...")
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		log.Fatal(err, "Failed to initialize view templates")
	}
	log.Info("View templates initialized.")

	// Initialize Layers and Middleware
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService, viewService, log)
	authHandler := handler.NewAuthHandler(authenticator, sessionManager)

	authzMiddleware := middleware.Authorizer(enforcer, sessionManager)
	errorMiddleware := middleware.Error(log, viewService)

	// Initialize Router
	router := handler.NewRouter(pageHandler, authHandler, authzMiddleware, errorMiddleware, sessionManager)

	// ... (server setup and graceful shutdown) ...
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

func seedDefaultPolicies(e *casbin.Enforcer, log logger.Logger) {
	log.Info("Seeding default authorization policies...")
	policies := [][]string{
		{"anonymous", "/view/*", "GET"},
		{"anonymous", "/auth/login", "GET"},
		{"anonymous", "/auth/callback", "GET"},
		{"editor", "/view/*", "GET"},
		{"editor", "/edit/*", "GET"},
		{"editor", "/save/*", "POST"},
	}
	for _, p := range policies {
		if has, _ := e.HasPolicy(p); !has {
			if _, err := e.AddPolicy(p); err != nil {
				log.Error(err, fmt.Sprintf("Failed to add policy %v", p))
			}
		}
	}
	if has, _ := e.HasRoleForUser("editor", "anonymous"); !has {
		if _, err := e.AddRoleForUser("editor", "anonymous"); err != nil {
			log.Error(err, "Failed to add role 'editor' -> 'anonymous'")
		}
	}
	log.Info("Policy seeding complete.")
}
