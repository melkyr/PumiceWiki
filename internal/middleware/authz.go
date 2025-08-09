package middleware

import (
	"context"
	"go-wiki-app/internal/auth"
	"net/http"

	"github.com/casbin/casbin/v2"
)

type contextKey string

const userContextKey contextKey = "user"

// UserInfo represents the essential user information extracted from the token.
type UserInfo struct {
	Subject string
	// We can add more fields here like Name, Email as needed.
}

// Authorizer creates a new middleware for authorization.
// It checks the user's permissions using Casbin.
func Authorizer(e *casbin.Enforcer, auth *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start with an anonymous user by default.
			userInfo := &UserInfo{Subject: "anonymous"}

			// Try to get the ID token from the session cookie.
			cookie, err := r.Cookie("id_token")
			if err == nil {
				// If a token exists, verify it.
				token, err := auth.IDTokenVerifier.Verify(r.Context(), cookie.Value)
				if err == nil {
					// If the token is valid, update the user info with the subject from the token.
					userInfo.Subject = token.Subject
				}
			}

			// Add the user info (either the real user or "anonymous") to the request context.
			ctx := context.WithValue(r.Context(), userContextKey, userInfo)
			r = r.WithContext(ctx)

			// Use Casbin to enforce the policy.
			// Format: enforcer.Enforce(subject, object, action)
			// Example: enforcer.Enforce("alice", "/view/some-page", "GET")
			allowed, err := e.Enforce(userInfo.Subject, r.URL.Path, r.Method)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// If allowed, proceed to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserInfo retrieves the user information from the request context.
// This is a helper function for handlers to know who the current user is.
func GetUserInfo(ctx context.Context) *UserInfo {
	if userInfo, ok := ctx.Value(userContextKey).(*UserInfo); ok {
		return userInfo
	}
	// This should not happen if the middleware is applied correctly.
	return &UserInfo{Subject: "anonymous"}
}
