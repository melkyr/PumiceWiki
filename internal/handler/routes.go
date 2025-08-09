package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new chi router.
func NewRouter(pageHandler *PageHandler, authHandler *AuthHandler, authzMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Home", http.StatusFound)
	})

	// Authentication routes
	r.Get("/auth/login", authHandler.handleLogin)
	r.Get("/auth/callback", authHandler.handleCallback)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authzMiddleware)

		r.Get("/view/{title}", pageHandler.viewHandler)
		r.Get("/edit/{title}", pageHandler.editHandler)
		r.Post("/save/{title}", pageHandler.saveHandler)
	})

	return r
}
