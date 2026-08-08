// Package yahoo is a thin client for the Yahoo Fantasy Sports API.
//
// Yahoo Fantasy data is private to a league member, so it uses 3-legged OAuth2:
// a one-time authorization produces a long-lived refresh token, after which the
// client mints access tokens automatically (client id + secret + refresh token).
package yahoo

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	authURL  = "https://api.login.yahoo.com/oauth2/request_auth"
	tokenURL = "https://api.login.yahoo.com/oauth2/get_token"

	// fantasyReadScope is Yahoo's read scope for Fantasy Sports.
	fantasyReadScope = "fspt-r"
)

// OAuthConfig builds the OAuth2 config for the Yahoo Fantasy API. redirectURI
// must exactly match the Redirect URI registered on the Yahoo developer app.
func OAuthConfig(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{fantasyReadScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

// AuthCodeURL returns the URL the user visits to authorize the app. state is an
// opaque value echoed back; for the manual paste-the-code flow it isn't verified.
func AuthCodeURL(cfg *oauth2.Config, state string) string {
	return cfg.AuthCodeURL(state)
}

// Exchange swaps an authorization code for a token (including the refresh token).
func Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	return cfg.Exchange(ctx, code)
}

// HTTPClient returns an *http.Client that authenticates every request with the
// refresh token, refreshing the short-lived access token automatically.
func HTTPClient(ctx context.Context, cfg *oauth2.Config, refreshToken string) *http.Client {
	return cfg.Client(ctx, &oauth2.Token{RefreshToken: refreshToken})
}
