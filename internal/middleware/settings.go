package middleware

import (
	"context"
	"go-wiki-app/internal/view"
	"net/http"
	"strings"
)

// legacyUserAgents contains substrings of User-Agent headers for browsers
// that are known to not support JavaScript or HTMX well.
var legacyUserAgents = []string{
	"Dillo",      // A graphical web browser known for its speed and small footprint.
	"Lynx",       // A classic text-based web browser.
	"w3m",        // Another popular text-based web browser.
	"NetSurf",    // A lightweight open-source browser with its own layout engine.
	"AmigaVoyager", // Web browser for AmigaOS.
	"Amiga-AWeb", // Another web browser for AmigaOS.
	"IBrowse",    // A web browser for AmigaOS.
}

// isLegacyBrowser checks if the User-Agent string matches known legacy browsers.
func isLegacyBrowser(userAgent string) bool {
	for _, ua := range legacyUserAgents {
		if strings.Contains(userAgent, ua) {
			return true
		}
	}
	return false
}

// SettingsMiddleware checks for a "basic=true" query parameter or a legacy browser
// User-Agent and sets a corresponding flag in the request context. This allows
// downstream handlers and templates to disable features like HTMX for a simpler,
// basic HTML experience.
func SettingsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start with the manual query parameter check.
		basicMode := r.URL.Query().Get("basic") == "true"

		// If not manually set, check the User-Agent for legacy browsers.
		if !basicMode {
			userAgent := r.Header.Get("User-Agent")
			basicMode = isLegacyBrowser(userAgent)
		}

		ctx := context.WithValue(r.Context(), view.BasicModeKey, basicMode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
