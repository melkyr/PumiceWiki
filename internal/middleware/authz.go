package middleware

import (
	"go-wiki-app/internal/session"
	"net/http"

	"github.com/casbin/casbin/v2"
)

// Authorizer is a middleware that enforces access control using Casbin.
// It performs the following steps:
// 1. Determines the user's subject from the session, defaulting to "anonymous".
// 2. Fetches the user's roles and display name and adds them to the request context.
// 3. Uses the Casbin enforcer to check if the subject is allowed to perform the
//    requested action (e.g., GET) on the requested resource (e.g., /view/SomePage).
// 4. If allowed, it passes the request to the next handler.
// 5. If not allowed, it returns a 403 Forbidden error.
func Authorizer(e casbin.IEnforcer, sm session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Identify the user (subject) from the session.
			subject := sm.GetString(r.Context(), "user_subject")
			if subject == "" {
				subject = "anonymous"
			}

			// 2. Enrich the request context with user information.
			roles, err := e.GetRolesForUser(subject)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				return
			}
			displayName := sm.GetString(r.Context(), "user_display_name")

			userInfo := &UserInfo{Subject: subject, Roles: roles, DisplayName: displayName}
			ctx := SetUserInfo(r.Context(), userInfo)
			r = r.WithContext(ctx)

			// 3. Enforce the policy using Casbin.
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
