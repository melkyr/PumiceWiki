package auth

import (
	"context"
	"go-wiki-app/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Authenticator is a struct that holds the OIDC provider, OAuth2 config, and ID token verifier.
type Authenticator struct {
	*oidc.Provider
	*oauth2.Config
	*oidc.IDTokenVerifier
}

// NewAuthenticator creates a new Authenticator by setting up the OIDC provider
// and OAuth2 configuration based on the application's config.
func NewAuthenticator(cfg *config.OIDCConfig) (*Authenticator, error) {
	// Use the OIDC discovery endpoint to get the provider configuration.
	provider, err := oidc.NewProvider(context.Background(), cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	// Create an OIDC ID token verifier.
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	// Create a new OAuth2 config with the credentials and endpoints from the provider.
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &Authenticator{
		Provider:        provider,
		Config:          oauth2Config,
		IDTokenVerifier: verifier,
	}, nil
}
