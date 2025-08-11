package auth

import (
	"context"
	"go-wiki-app/internal/config"
	"net"
	"net/http"
	"strings"

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
	// Create a custom HTTP client to handle the address translation.
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// When the OIDC client tries to connect to "localhost:8000" (the public-facing
				// address of Casdoor), we intercept it and change the address to "casdoor:8000"
				// (the internal Docker network address).
				if strings.HasPrefix(addr, "localhost:8000") {
					addr = "casdoor:8000"
				}
				return net.Dial(network, addr)
			},
		},
	}
	ctx := oidc.ClientContext(context.Background(), client)

	// Use the OIDC discovery endpoint to get the provider configuration.
	// We pass the custom client's context here.
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
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
