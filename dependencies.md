# Project Dependencies

This file documents the external Go modules that this project directly depends on, as listed in `go.mod`.

## Core Application

-   **`github.com/go-chi/chi/v5`**: A lightweight and idiomatic HTTP router for Go. Used to route incoming web requests to the correct handlers. We also use its middleware for logging, request IDs, and response compression.
-   **`github.com/jmoiron/sqlx`**: A set of extensions on the standard `database/sql` package. It simplifies database operations, especially scanning rows into structs. Used for both the main MariaDB database and the SQLite cache.
-   **`github.com/go-sql-driver/mysql`**: The database driver for MariaDB, used for all primary application data.
-   **`modernc.org/sqlite`**: A pure Go SQLite driver. Used for the persistent caching layer (`internal/cache`) to reduce load on the main database.
-   **`github.com/spf13/viper`**: A complete configuration solution for Go applications. Used to manage configuration from files (`config.yml`) and environment variables.

## Authentication & Authorization

-   **`github.com/coreos/go-oidc/v3/oidc`**: A client library for OpenID Connect (OIDC). Used to handle the authentication flow with Casdoor.
-   **`golang.org/x/oauth2`**: A core library for OAuth2 flows, which is a dependency for the OIDC library.
-   **`github.com/casbin/casbin/v2`**: The core library for authorization (access control). It allows us to define and enforce permissions based on roles.
-   **`github.com/memwey/casbin-sqlx-adapter`**: A Casbin adapter that allows storing authorization policies in our main MariaDB database using `sqlx`.

## Session Management

-   **`github.com/alexedwards/scs/v2`**: A modern and secure session management library for Go.
-   **`github.com/alexedwards/scs/mysqlstore`**: The MySQL/MariaDB storage engine for the `scs` session manager, allowing sessions to be persisted in the main database.

## Security & Data Handling

-   **`github.com/microcosm-cc/bluemonday`**: A fast and powerful HTML sanitizer. Used to clean user-provided content to prevent Cross-Site Scripting (XSS) attacks.
-   **`github.com/yuin/goldmark`**: A fast and extensible Markdown parser. Used to convert user-written Markdown into HTML for rendering in the browser.

## Database Migrations

-   **`github.com/golang-migrate/migrate/v4`**: A library for handling database schema migrations against the main MariaDB database.

## Logging

-   **`github.com/rs/zerolog`**: A high-performance, structured logging library. It allows for configurable log levels and formats (JSON/console) for better observability.
