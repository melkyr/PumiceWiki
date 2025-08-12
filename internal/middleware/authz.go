package middleware

import (
	"go-wiki-app/internal/session"
	"log"
	"net/http"

	"github.com/casbin/casbin/v2"
)

// Authorizer creates a new middleware for authorization.
// It checks the user's permissions using Casbin based on session data.
func Authorizer(e *casbin.Enforcer, sm session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject := sm.GetString(r.Context(), "user_subject")
			if subject == "" {
				subject = "anonymous"
			}

			roles, err := e.GetRolesForUser(subject)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				return
			}
			displayName := sm.GetString(r.Context(), "user_display_name")
			log.Printf("DEBUG: Authorizer middleware. Subject: '%s', displayName from session: '%s'", subject, displayName)

			userInfo := &UserInfo{Subject: subject, Roles: roles, DisplayName: displayName}
			ctx := SetUserInfo(r.Context(), userInfo)
			r = r.WithContext(ctx)

			allowed, err := e.Enforce(subject, r.URL.Path, r.Method)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
