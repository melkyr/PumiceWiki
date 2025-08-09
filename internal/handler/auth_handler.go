package handler

import (
	"crypto/rand"
	"encoding/base64"
	"go-wiki-app/internal/auth"
	"io"
	"net/http"
	"time"
)

// AuthHandler holds the dependencies for the authentication handlers.
type AuthHandler struct {
	auth *auth.Authenticator
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(a *auth.Authenticator) *AuthHandler {
	return &AuthHandler{auth: a}
}

// handleLogin redirects the user to the OIDC provider to log in.
// It uses a random 'state' string for CSRF protection.
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randString(16)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Store the state in a short-lived cookie to verify on callback.
	http.SetCookie(w, &http.Cookie{
		Name:     "state",
		Value:    state,
		Path:     "/",
		MaxAge:   int(10 * time.Minute / time.Second),
		HttpOnly: true,
		Secure:   r.TLS != nil,
	})
	http.Redirect(w, r, h.auth.AuthCodeURL(state), http.StatusFound)
}

// handleCallback is the redirect URL for the OIDC provider.
// It handles the code exchange and token verification.
func (h *AuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify the state parameter to prevent CSRF attacks.
	stateCookie, err := r.Cookie("state")
	if err != nil {
		http.Error(w, "state cookie not found", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}

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
	// The OIDC library internally checks the nonce, issuer, audience, and expiry.
	_, err = h.auth.IDTokenVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store the raw ID Token in a session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "id_token",
		Value:    rawIDToken,
		Path:     "/",
		MaxAge:   int(24 * time.Hour / time.Second),
		HttpOnly: true,
		Secure:   r.TLS != nil,
	})

	// Redirect user to the home page after successful login.
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
