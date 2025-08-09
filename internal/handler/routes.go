package handler

import (
	"io/fs"
	"net/http"

	"go-wiki-app/internal/middleware"
	"go-wiki-app/web"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new chi router.
func NewRouter(
	pageHandler *PageHandler,
	authHandler *AuthHandler,
	authzMiddleware func(http.Handler) http.Handler,
	errorMiddleware func(middleware.AppHandler) http.Handler, // New dependency
	sessionManager *scs.SessionManager,
) *chi.Mux {
	r := chi.NewRouter()

	// Base middleware stack
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(sessionManager.LoadAndSave)
	// The recoverer middleware is now handled by our custom error middleware,
	// but we can leave chi's here as a fallback.

	// Static File Server
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Home", http.StatusFound)
	})

	// Authentication Routes
	r.Group(func(r chi.Router) {
		if authHandler != nil {
			// Auth handlers are standard http.Handlers, not appHandlers,
			// so they are not wrapped in the error middleware.
			r.Get("/auth/login", authHandler.handleLogin)
			r.Get("/auth/callback", authHandler.handleCallback)
			r.Get("/auth/logout", authHandler.handleLogout)
		}
	})

	// Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(authzMiddleware)
		// Wrap each appHandler with the error middleware to convert it to an http.Handler.
		r.Method("GET", "/view/{title}", errorMiddleware(pageHandler.viewHandler))
		r.Method("GET", "/edit/{title}", errorMiddleware(pageHandler.editHandler))
		r.Method("POST", "/save/{title}", errorMiddleware(pageHandler.saveHandler))
	})

	return r
}
