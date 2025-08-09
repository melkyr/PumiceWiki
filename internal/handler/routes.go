package handler

import (
	"io/fs"
	"net/http"

	"go-wiki-app/web"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new chi router.
func NewRouter(
	pageHandler *PageHandler,
	authHandler *AuthHandler,
	authzMiddleware func(http.Handler) http.Handler,
	sessionManager *scs.SessionManager,
) *chi.Mux {
	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Add the session manager middleware.
	// This must come before any handlers that need session data.
	r.Use(sessionManager.LoadAndSave)

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
			r.Get("/auth/login", authHandler.handleLogin)
			r.Get("/auth/callback", authHandler.handleCallback)
		}
	})

	// Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(authzMiddleware)
		r.Get("/view/{title}", pageHandler.viewHandler)
		r.Get("/edit/{title}", pageHandler.editHandler)
		r.Post("/save/{title}", pageHandler.saveHandler)
	})

	return r
}
