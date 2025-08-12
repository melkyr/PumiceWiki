package handler

import (
	"crypto/rand"
	"encoding/base64"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/session"
	"io"
	"net/http"

	"github.com/casbin/casbin/v2"
)

// AuthHandler holds the dependencies for the authentication handlers.
type AuthHandler struct {
	auth    *auth.Authenticator
	session session.Manager
	enforcer *casbin.Enforcer
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(a *auth.Authenticator, sm session.Manager, e *casbin.Enforcer) *AuthHandler {
	return &AuthHandler{
		auth:    a,
		session: sm,
		enforcer: e,
	}
}

// handleLogin redirects the user to the OIDC provider to log in.
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randString(16)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.session.Put(r.Context(), "state", state)

	http.Redirect(w, r, h.auth.AuthCodeURL(state), http.StatusFound)
}

// handleCallback is the OIDC callback endpoint. It handles the authorization code,
// exchanges it for tokens, verifies the ID token, and establishes a user session.
func (h *AuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	// 1. Verify the state parameter to prevent CSRF attacks.
	state := h.session.GetString(r.Context(), "state")
	if state == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}
	h.session.Remove(r.Context(), "state")

	// 2. Exchange the authorization code for an OAuth2 token.
	oauth2Token, err := h.auth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Extract and verify the ID Token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}
	idToken, err := h.auth.IDTokenVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Parse custom claims from the ID Token.
	// We expect the OIDC provider (e.g., Casdoor) to be configured to send these claims.
	var claims struct {
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`
		Roles       []struct {
			Name string `json:"name"`
		} `json:"roles"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Synchronize user roles with Casbin.
	// This ensures that the user's permissions are always up-to-date with the OIDC provider.
	// First, remove any existing roles for this user to handle role changes.
	h.enforcer.DeleteRolesForUser(idToken.Subject)
	// Then, grant the new roles from the token.
	for _, role := range claims.Roles {
		h.enforcer.AddRoleForUser(idToken.Subject, role.Name)
	}

	// 6. Establish the user's session.
	// Determine the best display name to use, falling back from displayName to name.
	var displayName string
	if claims.DisplayName != "" {
		displayName = claims.DisplayName
	} else {
		displayName = claims.Name
	}
	h.session.Put(r.Context(), "raw_id_token", rawIDToken)
	h.session.Put(r.Context(), "user_subject", idToken.Subject)
	h.session.Put(r.Context(), "user_display_name", displayName)

	http.Redirect(w, r, "/", http.StatusFound)
}

// handleLogout destroys the user's session and redirects to the home page.
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	h.session.Destroy(r.Context())
	http.Redirect(w, r, "/", http.StatusFound)
}

// randString is a helper function to generate a random string for the 'state' parameter.
func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
