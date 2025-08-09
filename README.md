# Go Wiki Application

A lightweight, fast, and secure wiki application built with Go.

## Features

- **Page Management:** Create, view, and edit wiki pages.
- **Authentication:** Secure user authentication via OIDC with Casdoor.
- **Authorization:** Role-based access control (RBAC) using Casbin.
- **Fast Frontend:** Lightweight frontend built with HTMX for near-instant page loads.
- **Containerized:** Fully containerized with Docker and Docker Compose for easy deployment.

## Running with Docker Compose

This application is designed to be run with Docker Compose, which orchestrates the Go application and a Casdoor identity provider.

### Prerequisites

- Docker and Docker Compose installed.
- A `.env` file or environment variables for OIDC client secrets.

### Configuration

1.  **Casdoor Setup:**
    - After running `docker-compose up` for the first time, Casdoor will be available at `http://localhost:8000`.
    - Log in with the default credentials (`admin`/`casdoor`).
    - Create a new Application and a new User.
    - Note the `Client ID` and `Client Secret` for your application.

2.  **Application Setup:**
    - Update the `docker-compose.yml` file with your Casdoor `Client ID` and `Client Secret`:
      ```yaml
      environment:
        - WIKI_OIDC_CLIENT_ID=YOUR_CLIENT_ID
        - WIKI_OIDC_CLIENT_SECRET=YOUR_CLIENT_SECRET
      ```

### Running the Application

1.  Start the application stack:
    ```bash
    docker-compose up --build
    ```

2.  The wiki will be available at `http://localhost:8080`.

## Default Roles & Permissions

The application seeds the database with a default set of roles and permissions on startup:

- **`anonymous`**:
  - Can view all pages (`/view/*`).
  - Can access the login and callback routes.
- **`editor`**:
  - Can do everything an `anonymous` user can.
  - Can edit and save all pages (`/edit/*`, `/save/*`).

To assign a user to the `editor` role, you must do so manually within the Casdoor UI and ensure the corresponding policy is added to the Casbin database.
