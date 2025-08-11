package middleware

import (
	"context"
	"go-wiki-app/internal/view"
	"net/http"
)

// SettingsMiddleware checks for a "basic=true" query parameter and sets a corresponding
// flag in the request context. This allows downstream handlers and templates to
// disable features like HTMX for a simpler, basic HTML experience.
func SettingsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		basicMode := r.URL.Query().Get("basic") == "true"
		ctx := context.WithValue(r.Context(), view.BasicModeKey, basicMode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
