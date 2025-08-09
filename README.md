# Go Wiki Application

A lightweight, fast, and secure wiki application built with Go, HTMX, and Casdoor.

## Features

- **Page Management:** Create, view, and edit wiki pages.
- **Authentication:** Secure user authentication via OIDC with Casdoor.
- **Authorization:** Role-based access control (RBAC) using Casbin.
- **Fast Frontend:** Lightweight server-rendered frontend using Go Templates and HTMX.
- **Containerized:** Fully containerized with Docker and Docker Compose for easy deployment.
- **Structured Logging:** Configurable, structured logging with `zerolog`.
- **TLS Support:** Optional TLS/HTTPS support.

## Architecture Flow

The following diagram illustrates the request flow through the application:

```
+--------------+   1. HTTP Request   +---------------------------------------------+
|              |-------------------->| Chi Router                                  |
| Web Browser  |                     |   - Logging, Recovery Middleware            |
|              |                     |   - /static/* -> Static File Server         |
+--------------+                     |   - /auth/* -> Auth Handlers                |
      ^                              |   - /view/* -> Authz Middleware -> Page Handlers|
      |                              +---------------------------------------------+
      |                                                 |
   5. HTML Page                                         | 2. Authz Middleware
      |                                                 V
      |                              +---------------------------------------------+
      +----------------------------| Page/Auth Handlers                          |
                                   |   - Calls OIDC Authenticator (for login)    |
                                   |   - Calls Page Service (for wiki pages)     |
                                   |   - Calls View Service (to render HTML)     |
                                   +---------------------------------------------+
                                                     |         |
                                  3. Service Logic   |         | 4. Render Template
                                                     V         V
                                   +-----------------+---------+-------------------+
                                   | Page Service            | View Service      |
                                   | - Calls Repository      | - Executes HTML   |
                                   |                         |   Templates       |
                                   +-----------------+---------+-------------------+
                                                     |
                                                     V
                                   +-----------------+-----------------------------+
                                   | Page Repository         | Casbin Enforcer   |
                                   | - Queries SQLite DB     | - Queries Policies|
                                   +---------------------------------------------+

```

## Getting Started

This application is designed to be run with Docker Compose, which orchestrates the Go application and a Casdoor identity provider.

### 1. Prerequisites

- Docker and Docker Compose must be installed on your system.

### 2. Configuration

The application is configured via environment variables, which can be set directly in the `docker-compose.yml` file or loaded from a `.env` file.

**A. Casdoor Setup:**

Before you can log in to the wiki, you need to configure Casdoor and get OIDC client credentials.

1.  Run `docker-compose up` once to start the Casdoor service.
2.  Navigate to `http://localhost:8000` in your browser.
3.  Log in with the default credentials: `admin` / `casdoor`.
4.  In the Casdoor UI, create a new **Application**.
    -   Note the `Client ID` and `Client Secret`.
5.  Ensure the **Redirect URL** in your Casdoor application settings is set to `http://localhost:8080/auth/callback`.

**B. Wiki Application Setup:**

1.  Open the `docker-compose.yml` file.
2.  Find the `environment` section for the `app` service.
3.  Replace the placeholder values for `WIKI_OIDC_CLIENT_ID` and `WIKI_OIDC_CLIENT_SECRET` with the credentials you obtained from Casdoor.

### 3. Running the Application

1.  Start the application stack from the root of the project:
    ```bash
    docker-compose up --build
    ```
    The `--build` flag is only necessary the first time or after making code changes.

2.  The wiki application will be available at `http://localhost:8080`.

## Configuration Details

All configuration can be set via environment variables, which override the defaults in `config.yml`.

| Variable                      | Description                                           | Default                  |
| ----------------------------- | ----------------------------------------------------- | ------------------------ |
| `WIKI_SERVER_PORT`            | Port for the wiki server to listen on.                | `8080`                   |
| `WIKI_SERVER_TLS_ENABLED`     | Set to `true` to enable HTTPS.                        | `false`                  |
| `WIKI_SERVER_TLS_CERTFILE`    | Path to the TLS certificate file.                     | `cert.pem`               |
| `WIKI_SERVER_TLS_KEYFILE`     | Path to the TLS key file.                             | `key.pem`                |
| `WIKI_DB_DSN`                 | Data Source Name for the database.                    | `wiki.db`                |
| `WIKI_OIDC_ISSUER_URL`        | The issuer URL of your OIDC provider.                 | `http://casdoor:8000`    |
| `WIKI_OIDC_CLIENT_ID`         | The client ID for the OIDC application.               | `YOUR_CLIENT_ID`         |
| `WIKI_OIDC_CLIENT_SECRET`     | The client secret for the OIDC application.           | `YOUR_CLIENT_SECRET`     |
| `WIKI_OIDC_REDIRECT_URL`      | The callback URL for OIDC.                            | `http://localhost:8080/auth/callback` |
| `WIKI_LOG_LEVEL`              | The logging level (`debug`, `info`, `warn`, `error`). | `info`                   |
| `WIKI_LOG_FORMAT`             | The log format (`console` or `json`).                 | `console`                |

## Default Roles & Permissions

The application seeds the database with a default set of roles and permissions on startup:

- **`anonymous`**:
  - Can view all pages (`/view/*`).
  - Can access the login and callback routes (`/auth/*`).
- **`editor`**:
  - Inherits all permissions from `anonymous`.
  - Can access the edit form for all pages (`/edit/*`).
  - Can save pages (`/save/*`).

To assign a user to the `editor` role, you must do so manually within Casdoor and ensure the corresponding policy is added to the Casbin database. This part of the workflow is currently manual.

## Developer Workflow: Modifying Static Assets

This project uses Go's `embed` package to bundle all static assets (CSS, JS) and HTML templates directly into the application binary. This creates a single, self-contained executable, which simplifies deployment.

The important consequence of this design is that **you cannot change static assets on a running server.** To modify any CSS, JavaScript, or HTML templates, you must:

1.  Edit the source files in the `/web/static` or `/web/templates` directories in your local development environment.
2.  Rebuild the application's Docker image. The provided `Dockerfile` handles the embedding process.
3.  Deploy the new Docker image.

If you are using the provided `docker-compose.yml`, the workflow is:

```bash
# After making changes to files in /web/...
docker-compose up --build
```

The `--build` flag tells Docker Compose to rebuild the `app` image, which will include your updated assets.
