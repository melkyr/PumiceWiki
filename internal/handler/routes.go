package handler

import (
	"io/fs"
	"net/http"

	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/session"
	"go-wiki-app/web"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new chi router.
func NewRouter(
	pageHandler *PageHandler,
	authHandler *AuthHandler,
	seoHandler *SeoHandler,
	authzMiddleware func(http.Handler) http.Handler,
	errorMiddleware func(middleware.AppHandler) http.Handler,
	sessionManager session.Manager,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Compress(5))
	r.Use(sessionManager.LoadAndSave)
	r.Use(middleware.SettingsMiddleware)

	staticFS, _ := fs.Sub(web.StaticFS, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// SEO routes
	r.Get("/robots.txt", seoHandler.robotsHandler)
	r.Get("/sitemap.xml", seoHandler.sitemapHandler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Home", http.StatusFound)
	})

	r.Group(func(r chi.Router) {
		if authHandler != nil {
			r.Get("/auth/login", authHandler.handleLogin)
			r.Get("/auth/callback", authHandler.handleCallback)
			r.Get("/auth/logout", authHandler.handleLogout)
		}
	})

	r.Group(func(r chi.Router) {
		r.Use(authzMiddleware)
		r.Method("GET", "/view/{title}", errorMiddleware(pageHandler.viewHandler))
		r.Method("GET", "/edit/{title}", errorMiddleware(pageHandler.editHandler))
		r.Method("POST", "/save/{title}", errorMiddleware(pageHandler.saveHandler))
		r.Method("GET", "/list", errorMiddleware(pageHandler.listHandler))
		r.Method("GET", "/categories", errorMiddleware(pageHandler.categoriesHandler))
		r.Method("GET", "/api/search/categories", errorMiddleware(pageHandler.searchCategoriesHandler))
		r.Method("GET", "/category/{categoryName}", errorMiddleware(pageHandler.viewByCategoryHandler))
		r.Method("GET", "/category/{categoryName}/{subcategoryName}", errorMiddleware(pageHandler.viewBySubcategoryHandler))
	})

	return r
}
