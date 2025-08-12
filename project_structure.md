# Project Structure

This document outlines the directory structure of the Go Wiki application, explaining the purpose of each major file and folder.

-   `cmd/`: Application entry points.
    -   `cmd/server/main.go`: The main entry point for the web server. It initializes all dependencies and starts the HTTP server.
-   `internal/`: Contains the core application logic, structured by domain.
    -   `internal/auth/`: Handles authentication (OIDC) and authorization (Casbin) logic.
    -   `internal/config/`: Manages application configuration, loading from files and environment variables.
    -   `internal/data/`: Responsible for database interactions, including models (`models.go`) and the repository layer (`page_repository.go`).
    -   `internal/handler/`: Contains the HTTP handlers that respond to web requests (e.g., `page_handler.go`, `auth_handler.go`). It maps routes to business logic and configures middleware from `chi` and the `internal/middleware` package.
    -   `internal/middleware/`: Implements HTTP middleware for tasks like logging, authentication checks, and setting request-scoped values.
    -   `internal/service/`: Contains the core business logic of the application (e.g., `page_service.go`). Handlers call services to perform actions.
    -   `internal/session/`: Manages user sessions.
    -   `internal/view/`: Handles the rendering of HTML templates.
-   `web/`: Contains all frontend assets.
    -   `web/static/`: Static files like CSS and images.
    -   `web/templates/`: HTML templates used for rendering pages.
        -   `layouts/`: Base layout templates.
        -   `pages/`: Templates for specific pages (view, edit, etc.).
    -   `web/embed.go`: Uses Go's `embed` package to bundle the `static` and `templates` directories into the application binary.
-   `migrations/`: SQL database migration files.
-   `docker/`: Contains Docker-related files, such as `init.sql` for the database.
-   `config.yml`: Default configuration file.
-   `docker-compose.yml`: Defines the services, networks, and volumes for the Docker application stack.
-   `go.mod`, `go.sum`: Go module files for managing project dependencies.
