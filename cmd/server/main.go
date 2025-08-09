package main

import (
	"fmt"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/handler"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"log"
	"net/http"
	"os"
)

func main() {
	// 1. Initialize Logger
	logger := log.New(os.Stdout, "WIKI_APP ", log.LstdFlags|log.Lshortfile)

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// 3. Apply Migrations
	logger.Println("Applying database migrations...")
	if err := data.ApplyMigrations(cfg.DB.DSN, "migrations"); err != nil {
		logger.Fatalf("Failed to apply migrations: %v", err)
	}
	logger.Println("Migrations applied successfully.")

	// 4. Initialize Database
	logger.Println("Connecting to the database...")
	db, err := data.NewDB(cfg.DB.DSN)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	logger.Println("Database connection successful.")

	// 5. Initialize Auth Components
	logger.Println("Initializing authentication and authorization...")
	authenticator, err := auth.NewAuthenticator(&cfg.OIDC)
	if err != nil {
		logger.Fatalf("Failed to initialize authenticator: %v", err)
	}
	enforcer, err := auth.NewEnforcer("sqlite3", cfg.DB.DSN, "auth_model.conf")
	if err != nil {
		logger.Fatalf("Failed to initialize enforcer: %v", err)
	}
	logger.Println("Auth components initialized.")

	// 6. Initialize Layers (Repo -> Service -> Handler)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService, logger)
	authHandler := handler.NewAuthHandler(authenticator)

	// 7. Initialize Router
	authzMiddleware := middleware.Authorizer(enforcer, authenticator)
	router := handler.NewRouter(pageHandler, authHandler, authzMiddleware)

	// 8. Start HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Printf("Starting server on %s", serverAddr)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", serverAddr, err)
	}
}
