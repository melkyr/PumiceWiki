package middleware

import "context"

// contextKey defines a custom type for context keys to avoid collisions.
type contextKey string

const userContextKey = contextKey("user")

// UserInfo represents the essential user information stored in the session and request context.
type UserInfo struct {
	Subject string
	Roles   []string
}

// GetUserInfo retrieves the user information from the request context.
func GetUserInfo(ctx context.Context) *UserInfo {
	if userInfo, ok := ctx.Value(userContextKey).(*UserInfo); ok {
		return userInfo
	}
	// Return an anonymous user if no user info is found in the context.
	return &UserInfo{Subject: "anonymous"}
}

// SetUserInfo adds the user information to the request context.
func SetUserInfo(ctx context.Context, userInfo *UserInfo) context.Context {
	return context.WithValue(ctx, userContextKey, userInfo)
}
