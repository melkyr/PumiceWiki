package middleware

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/casbin/casbin/v2"
)

type contextKey string

const userContextKey contextKey = "user"

// UserInfo represents the essential user information stored in the session.
type UserInfo struct {
	Subject string
}

// Authorizer creates a new middleware for authorization.
// It checks the user's permissions using Casbin based on session data.
func Authorizer(e *casbin.Enforcer, sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the user's subject from the session.
			// If not present, it will be an empty string.
			subject := sm.GetString(r.Context(), "user_subject")
			if subject == "" {
				subject = "anonymous"
			}

			// Add user info to the request context for downstream handlers.
			userInfo := &UserInfo{Subject: subject}
			ctx := context.WithValue(r.Context(), userContextKey, userInfo)
			r = r.WithContext(ctx)

			// Use Casbin to enforce the policy.
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

// GetUserInfo retrieves the user information from the request context.
func GetUserInfo(ctx context.Context) *UserInfo {
	if userInfo, ok := ctx.Value(userContextKey).(*UserInfo); ok {
		return userInfo
	}
	return &UserInfo{Subject: "anonymous"}
}
