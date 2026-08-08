package yahoo

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

// tokenSettingKey is the settings key under which the Yahoo OAuth token is stored.
const tokenSettingKey = "yahoo_oauth_token"

// SettingsStore is the persistence the token layer needs. *store.DB satisfies it.
type SettingsStore interface {
	SetSetting(ctx context.Context, key, value string) error
	GetSetting(ctx context.Context, key string) (string, bool, error)
}

// SaveToken persists the OAuth token (including the refresh token) as JSON.
func SaveToken(ctx context.Context, settings SettingsStore, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return settings.SetSetting(ctx, tokenSettingKey, string(data))
}

// LoadToken loads the persisted OAuth token; the bool is false if none is stored.
func LoadToken(ctx context.Context, settings SettingsStore) (*oauth2.Token, bool, error) {
	raw, ok, err := settings.GetSetting(ctx, tokenSettingKey)
	if err != nil || !ok {
		return nil, ok, err
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return nil, false, fmt.Errorf("unmarshal token: %w", err)
	}
	return &token, true, nil
}

// TokenSource returns an oauth2.TokenSource seeded from the persisted token that
// refreshes access tokens automatically and writes any refreshed token back to
// the store (so a rotated refresh token is never lost). It errors if no token is
// stored yet — run `bot yahoo-auth` first.
func TokenSource(ctx context.Context, cfg *oauth2.Config, settings SettingsStore) (oauth2.TokenSource, error) {
	token, ok, err := LoadToken(ctx, settings)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no stored Yahoo token; run `bot yahoo-auth` first")
	}
	persisting := &persistingSource{
		base:     cfg.TokenSource(ctx, token),
		settings: settings,
		ctx:      ctx,
		lastRT:   token.RefreshToken,
	}
	return oauth2.ReuseTokenSource(token, persisting), nil
}

// persistingSource writes the token back to the store whenever the refresh token
// changes (i.e. Yahoo rotated it).
type persistingSource struct {
	base     oauth2.TokenSource
	settings SettingsStore
	ctx      context.Context //nolint:containedctx // TokenSource.Token has no ctx param
	lastRT   string
}

func (p *persistingSource) Token() (*oauth2.Token, error) {
	token, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	if token.RefreshToken != "" && token.RefreshToken != p.lastRT {
		if saveErr := SaveToken(p.ctx, p.settings, token); saveErr == nil {
			p.lastRT = token.RefreshToken
		}
	}
	return token, nil
}
