package middleware

import (
	"context"
	"net/http"
)

type settingsKey string

const (
	// BasicModeKey is the key for the basic mode setting in the request context.
	BasicModeKey settingsKey = "basicMode"
)

// SettingsMiddleware checks for a "basic=true" query parameter and sets a corresponding
// flag in the request context. This allows downstream handlers and templates to
// disable features like HTMX for a simpler, basic HTML experience.
func SettingsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		basicMode := r.URL.Query().Get("basic") == "true"
		ctx := context.WithValue(r.Context(), BasicModeKey, basicMode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsBasicMode returns true if the "basic mode" flag is set in the request context.
func IsBasicMode(ctx context.Context) bool {
	basic, ok := ctx.Value(BasicModeKey).(bool)
	return ok && basic
}
