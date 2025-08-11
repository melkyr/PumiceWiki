package session

import (
	"context"
	"net/http"
)

// Manager is an interface that abstracts the session management implementation.
// This allows for easier testing and dependency injection.
type Manager interface {
	LoadAndSave(next http.Handler) http.Handler
	Put(ctx context.Context, key string, val interface{})
	GetString(ctx context.Context, key string) string
	PopString(ctx context.Context, key string) string
	Destroy(ctx context.Context) error
	Remove(ctx context.Context, key string)
}
