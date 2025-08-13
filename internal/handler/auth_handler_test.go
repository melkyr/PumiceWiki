//go:build unit

package handler

import (
	"context"
	"go-wiki-app/internal/session"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockSessionManager is a mock implementation of the session.Manager interface.
type mockSessionManager struct {
	destroyCalled bool
	putKey        string
	putValue      interface{}
}

// Ensure mockSessionManager implements the session.Manager interface.
var _ session.Manager = (*mockSessionManager)(nil)

func (m *mockSessionManager) LoadAndSave(next http.Handler) http.Handler { return next }
func (m *mockSessionManager) Put(ctx context.Context, key string, val interface{}) {
	m.putKey = key
	m.putValue = val
}
func (m *mockSessionManager) GetString(ctx context.Context, key string) string   { return "" }
func (m *mockSessionManager) PopString(ctx context.Context, key string) string   { return "" }
func (m *mockSessionManager) Remove(ctx context.Context, key string)             {}
func (m *mockSessionManager) Destroy(ctx context.Context) error {
	m.destroyCalled = true
	return nil
}

func TestLogoutHandler(t *testing.T) {
	// Arrange
	mockSession := &mockSessionManager{}
	// We pass nil for the authenticator and enforcer as they are not used by the logout handler.
	authHandler := NewAuthHandler(nil, mockSession, nil)

	req := httptest.NewRequest("GET", "/auth/logout", nil)
	rr := httptest.NewRecorder()

	// Act
	authHandler.handleLogout(rr, req)

	// Assert
	if !mockSession.destroyCalled {
		t.Error("expected session.Destroy to be called, but it wasn't")
	}

	if rr.Code != http.StatusFound {
		t.Errorf("want status code %d; got %d", http.StatusFound, rr.Code)
	}

	location, err := rr.Result().Location()
	if err != nil {
		t.Fatalf("could not get redirect location: %v", err)
	}
	if location.Path != "/" {
		t.Errorf("want redirect to '/'; got '%s'", location.Path)
	}
}
