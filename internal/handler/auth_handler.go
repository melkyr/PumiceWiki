package handler

import (
	"crypto/rand"
	"encoding/base64"
	"go-wiki-app/internal/auth"
	"io"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

// AuthHandler holds the dependencies for the authentication handlers.
type AuthHandler struct {
	auth    *auth.Authenticator
	session *scs.SessionManager
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(a *auth.Authenticator, sm *scs.SessionManager) *AuthHandler {
	return &AuthHandler{
		auth:    a,
		session: sm,
	}
}

// handleLogin redirects the user to the OIDC provider to log in.
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randString(16)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Store the state in the session for CSRF protection.
	h.session.Put(r.Context(), "state", state)

	http.Redirect(w, r, h.auth.AuthCodeURL(state), http.StatusFound)
}

// handleCallback is the redirect URL for the OIDC provider.
func (h *AuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify the state parameter from the session.
	state := h.session.GetString(r.Context(), "state")
	if state == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}
	// State is single-use, so remove it.
	h.session.Remove(r.Context(), "state")

	// Exchange the authorization code for an OAuth2 token.
	oauth2Token, err := h.auth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract the ID Token from the OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	// Verify the ID Token's signature and claims.
	idToken, err := h.auth.IDTokenVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store user info in the session.
	h.session.Put(r.Context(), "raw_id_token", rawIDToken)
	h.session.Put(r.Context(), "user_subject", idToken.Subject)

	// Redirect user to the home page after successful login.
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleLogout destroys the user's session and redirects to the home page.
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Destroy the session and redirect.
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
