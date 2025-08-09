package main

import (
	"fmt"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/handler"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"log"
	"net/http"
	"os"

	"github.com/casbin/casbin/v2"
)

func main() {
	logger := log.New(os.Stdout, "WIKI_APP ", log.LstdFlags|log.Lshortfile)

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Println("Applying database migrations...")
	if err := data.ApplyMigrations(cfg.DB.DSN, "migrations"); err != nil {
		logger.Fatalf("Failed to apply migrations: %v", err)
	}
	logger.Println("Migrations applied successfully.")

	logger.Println("Connecting to the database...")
	db, err := data.NewDB(cfg.DB.DSN)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	logger.Println("Database connection successful.")

	logger.Println("Initializing authentication and authorization...")
	authenticator, err := auth.NewAuthenticator(&cfg.OIDC)
	if err != nil {
		logger.Fatalf("Failed to initialize authenticator: %v", err)
	}
	enforcer, err := auth.NewEnforcer("sqlite3", cfg.DB.DSN, "auth_model.conf")
	if err != nil {
		logger.Fatalf("Failed to initialize enforcer: %v", err)
	}
	seedDefaultPolicies(enforcer, logger)
	logger.Println("Auth components initialized and policies seeded.")

	logger.Println("Initializing view templates...")
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		logger.Fatalf("Failed to initialize view templates: %v", err)
	}
	logger.Println("View templates initialized.")

	// Initialize Layers (Repo -> Service -> Handler)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService, viewService, logger)
	authHandler := handler.NewAuthHandler(authenticator)

	// Initialize Router
	authzMiddleware := middleware.Authorizer(enforcer, authenticator)
	router := handler.NewRouter(pageHandler, authHandler, authzMiddleware)

	// Start HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	if cfg.Server.TLS.Enabled {
		logger.Printf("Starting HTTPS server on %s", serverAddr)
		err = server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
	} else {
		logger.Printf("Starting HTTP server on %s", serverAddr)
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not start server: %v\n", err)
	}
}

func seedDefaultPolicies(e *casbin.Enforcer, logger *log.Logger) {
	logger.Println("Seeding default authorization policies...")

	policies := [][]string{
		// anonymous role can view pages and login
		{"anonymous", "/view/*", "GET"},
		{"anonymous", "/auth/login", "GET"},
		{"anonymous", "/auth/callback", "GET"},
		// editor role can do everything anonymous can, plus edit and save
		{"editor", "/view/*", "GET"},
		{"editor", "/edit/*", "GET"},
		{"editor", "/save/*", "POST"},
	}

	for _, p := range policies {
		if has, _ := e.HasPolicy(p); !has {
			if _, err := e.AddPolicy(p); err != nil {
				logger.Printf("Failed to add policy %v: %v", p, err)
			}
		}
	}

	// editor role inherits anonymous permissions (though redundant here, it's good practice)
	if has, _ := e.HasRoleForUser("editor", "anonymous"); !has {
		if _, err := e.AddRoleForUser("editor", "anonymous"); err != nil {
			logger.Printf("Failed to add role 'editor' -> 'anonymous': %v", err)
		}
	}

	logger.Println("Policy seeding complete.")
}
