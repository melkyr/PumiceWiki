package main

import (
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

	"github.com/casbin/casbin/v2"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// Use a basic logger for config errors
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	// 1. Initialize Logger
	log := logger.New(cfg.Log)

	// 2. Apply Migrations
	log.Info("Applying database migrations...")
	if err := data.ApplyMigrations(cfg.DB.DSN, "migrations"); err != nil {
		log.Fatal(err, "Failed to apply migrations")
	}
	log.Info("Migrations applied successfully.")

	// 3. Initialize Database
	log.Info("Connecting to the database...")
	db, err := data.NewDB(cfg.DB.DSN)
	if err != nil {
		log.Fatal(err, "Failed to connect to database")
	}
	defer db.Close()
	log.Info("Database connection successful.")

	// 4. Initialize Auth Components
	log.Info("Initializing authentication and authorization...")
	authenticator, err := auth.NewAuthenticator(&cfg.OIDC)
	if err != nil {
		log.Fatal(err, "Failed to initialize authenticator")
	}
	enforcer, err := auth.NewEnforcer("sqlite3", cfg.DB.DSN, "auth_model.conf")
	if err != nil {
		log.Fatal(err, "Failed to initialize enforcer")
	}
	seedDefaultPolicies(enforcer, log)
	log.Info("Auth components initialized and policies seeded.")

	// 5. Initialize View Templates
	log.Info("Initializing view templates...")
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		log.Fatal(err, "Failed to initialize view templates")
	}
	log.Info("View templates initialized.")

	// 6. Initialize Layers (Repo -> Service -> Handler)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService, viewService, log)
	authHandler := handler.NewAuthHandler(authenticator)

	// 7. Initialize Router
	authzMiddleware := middleware.Authorizer(enforcer, authenticator)
	router := handler.NewRouter(pageHandler, authHandler, authzMiddleware)

	// 8. Start HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	if cfg.Server.TLS.Enabled {
		log.Info(fmt.Sprintf("Starting HTTPS server on %s", serverAddr))
		err = server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
	} else {
		log.Info(fmt.Sprintf("Starting HTTP server on %s", serverAddr))
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err, "Could not start server")
	}
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
