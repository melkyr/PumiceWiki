package handler

import (
	"io/fs"
	"net/http"

	"go-wiki-app/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new chi router.
func NewRouter(pageHandler *PageHandler, authHandler *AuthHandler, authzMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- Static File Server ---
	// Create a sub-filesystem that only contains the 'static' directory.
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	// Create a file server handler.
	fileServer := http.FileServer(http.FS(staticFS))
	// Mount the file server. We strip the prefix so that a request to "/static/css/style.css"
	// correctly looks for "/css/style.css" in the sub-filesystem.
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// --- Public Routes ---
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Home", http.StatusFound)
	})

	// --- Authentication Routes ---
	r.Group(func(r chi.Router) {
		// The auth routes should not be behind the main authorizer,
		// as anonymous users need to access them.
		if authHandler != nil {
			r.Get("/auth/login", authHandler.handleLogin)
			r.Get("/auth/callback", authHandler.handleCallback)
		}
	})

	// --- Protected Routes ---
	r.Group(func(r chi.Router) {
		r.Use(authzMiddleware)

		r.Get("/view/{title}", pageHandler.viewHandler)
		r.Get("/edit/{title}", pageHandler.editHandler)
		r.Post("/save/{title}", pageHandler.saveHandler)
	})

	return r
}
