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

### Basic HTML Mode for Legacy Browsers

This wiki is designed to be accessible to a wide range of web browsers, including older or text-based browsers that do not support JavaScript. To achieve this, the application can serve a "Basic HTML Mode."

In this mode:
- All HTMX and JavaScript-based enhancements are disabled.
- The rich markdown editor is replaced with a simple textarea.
- The application serves plain, semantic HTML, ensuring all content is accessible.

This mode is activated in two ways:

1.  **Automatic Detection**: The application automatically enables Basic HTML Mode for known legacy browsers by checking the `User-Agent` string. The current list of detected agents includes:
    -   `Dillo`
    -   `Lynx`
    -   `w3m`
    -   `NetSurf`
    -   `AmigaVoyager`
    -   `Amiga-AWeb`
    -   `IBrowse`

2.  **Manual Override**: You can force any browser into Basic HTML Mode by adding the `?basic=true` query parameter to the URL. For example: `http://localhost:8080/view/Home?basic=true`.

### Note on Home Page Content

The default content for the "Home" page, which is displayed when the page does not yet exist in the database, is hardcoded within the application. It is not sourced from an external template file. If you need to change this default message ("Welcome! This page is empty."), you can find it in `internal/handler/page_handler.go` inside the `viewHandler` function.

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
                                   | - Queries MariaDB       | - Queries Policies|
                                   +---------------------------------------------+

```

## Getting Started

This application is designed to be run with Docker Compose, which orchestrates the Go application and a Casdoor identity provider.

### 1. Prerequisites

- Docker and Docker Compose must be installed on your system.

### 2. Configuration

The application is configured via environment variables, which can be set directly in the `docker-compose.yml` file or loaded from a `.env` file.

**A. Local Development Hostname**

For the application and the Casdoor identity provider to communicate correctly in a local Docker environment, you need to map the hostname `casdoor.local` to your local machine's loopback address.

Add the following line to your `hosts` file:

```
127.0.0.1 casdoor.local
```

-   **On macOS or Linux:** The file is located at `/etc/hosts`.
-   **On Windows:** The file is located at `C:\Windows\System32\drivers\etc\hosts`.

This is a one-time setup that allows your browser and the Go application container to refer to the Casdoor service by the same name.

**B. Casdoor Setup:**

Before you can log in to the wiki, you need to configure Casdoor and get OIDC client credentials.

1.  Run `docker-compose up` once to start the Casdoor service.
2.  Navigate to `http://casdoor.local:8000` in your browser.
3.  Log in with the default credentials: `admin` / `casdoor`.
4.  In the Casdoor UI, create a new **Application**.
    -   Note the `Client ID` and `Client Secret`.
5.  Ensure the **Redirect URL** in your Casdoor application settings is set to `http://localhost:8080/auth/callback`.

**C. Wiki Application Setup:**

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
| `WIKI_DB_DSN`                 | Data Source Name for the MariaDB database.            | `wikiuser:wikipass@tcp(mariadb:3306)/go_wiki_app?parseTime=true` |
| `WIKI_OIDC_ISSUER_URL`        | The issuer URL of your OIDC provider.                 | `http://casdoor.local:8000` |
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

To assign a user to the `editor` role, you can now do so directly in the Casdoor UI. The user's roles will be automatically synchronized with the wiki application upon login.

### Creating an Editor User (Automated Workflow)

This application is configured to automatically synchronize user roles from the Casdoor identity provider. When a user logs in, their roles are read from the authentication token and applied in the wiki's authorization system.

To make a user an `editor`, you need to perform two steps in the Casdoor dashboard.

**Step 1: Configure Casdoor to Send Roles in the Token**

You need to tell Casdoor to include the user's roles in the ID Token it issues for the wiki application.

1.  Log in to the Casdoor dashboard at `http://casdoor.local:8000`.
2.  Navigate to **Applications** and select your wiki application for editing.
3.  Find the section for **Token Configuration** or **Claims**.
4.  Add a new claim mapping:
    -   **Claim Name:** `roles`
    -   **Token Type:** `ID Token`
    -   **Claim Value:** `user.roles` (This assumes Casdoor uses an expression like this to access the user's assigned roles. The exact value may differ depending on Casdoor's configuration options.)
5.  Save the changes.

**Step 2: Assign the 'editor' Role to a User**

Now you can assign the `editor` role to any user.

1.  In the Casdoor dashboard, navigate to **Users**.
2.  Select the user you want to make an editor.
3.  Find the **Roles** section for the user.
4.  Assign them the `editor` role. (You may need to create the `editor` role in the **Roles** section of Casdoor first if it doesn't exist).
5.  Save the changes.

Now, the next time this user logs into the wiki application, they will automatically be granted editor permissions.

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

## Testing Strategy

This project uses two types of tests: unit tests and integration tests.

### Unit Tests

- **Purpose:** To test individual functions or components (like the `PageService`) in complete isolation. They use mock dependencies and do not require a database or running server.
- **Location:** Files ending in `_unit_test.go`.
- **How to Run:**
  ```bash
  go test -v ./...
  ```

### Integration Tests

- **Purpose:** To test how multiple components work together. They use a real in-memory database and test the full HTTP request/response cycle.
- **Location:** Files ending in `_integration_test.go`, marked with a `//go:build integration` tag.
- **How to Run:**
  ```bash
  go test -v -tags=integration ./...
  ```
