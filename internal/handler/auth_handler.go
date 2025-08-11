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

// handleCallback is the redirect URL for the OIDC provider.
func (h *AuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	state := h.session.GetString(r.Context(), "state")
	if state == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}
	h.session.Remove(r.Context(), "state")

	oauth2Token, err := h.auth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

	// Define a struct to hold the custom claims, including roles.
	// We expect the OIDC provider (Casdoor) to be configured to send roles in this claim.
	var claims struct {
		Roles []struct {
			Name string `json:"name"`
		} `json:"roles"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// First, remove any existing roles for this user to handle role changes.
	h.enforcer.DeleteRolesForUser(idToken.Subject)

	// Grant the new roles from the token.
	for _, role := range claims.Roles {
		h.enforcer.AddRoleForUser(idToken.Subject, role.Name)
	}

	h.session.Put(r.Context(), "raw_id_token", rawIDToken)
	h.session.Put(r.Context(), "user_subject", idToken.Subject)

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
