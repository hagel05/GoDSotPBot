package yahoo

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
)

// fakeSettings is an in-memory SettingsStore.
type fakeSettings struct{ m map[string]string }

func newFakeSettings() *fakeSettings { return &fakeSettings{m: map[string]string{}} }

func (f *fakeSettings) SetSetting(_ context.Context, key, value string) error {
	f.m[key] = value
	return nil
}

func (f *fakeSettings) GetSetting(_ context.Context, key string) (string, bool, error) {
	v, ok := f.m[key]
	return v, ok, nil
}

func TestSaveLoadTokenRoundTrip(t *testing.T) {
	ctx := context.Background()
	settings := newFakeSettings()

	if err := SaveToken(ctx, settings, &oauth2.Token{RefreshToken: "rt-123", AccessToken: "at-abc"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	loaded, ok, err := LoadToken(ctx, settings)
	if err != nil || !ok {
		t.Fatalf("LoadToken: ok=%v err=%v", ok, err)
	}
	if loaded.RefreshToken != "rt-123" {
		t.Errorf("refresh token = %q, want rt-123", loaded.RefreshToken)
	}
}

func TestLoadTokenEmpty(t *testing.T) {
	_, ok, err := LoadToken(context.Background(), newFakeSettings())
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if ok {
		t.Error("expected ok=false on empty store")
	}
}

func TestTokenSourceRequiresStoredToken(t *testing.T) {
	cfg := OAuthConfig("id", "secret", "https://localhost/cb")
	if _, err := TokenSource(context.Background(), cfg, newFakeSettings()); err == nil {
		t.Error("expected error when no token is stored")
	}
}

// stubSource returns a fixed token, standing in for the refreshing base source.
type stubSource struct{ tok *oauth2.Token }

func (s stubSource) Token() (*oauth2.Token, error) { return s.tok, nil }

func TestPersistingSourceSavesRotatedToken(t *testing.T) {
	settings := newFakeSettings()
	// Seed an initial token so we can observe the rotation being written.
	_ = SaveToken(context.Background(), settings, &oauth2.Token{RefreshToken: "old-rt"})

	src := &persistingSource{
		base:     stubSource{tok: &oauth2.Token{RefreshToken: "new-rt", AccessToken: "fresh"}},
		settings: settings,
		ctx:      context.Background(),
		lastRT:   "old-rt",
	}
	if _, err := src.Token(); err != nil {
		t.Fatalf("Token: %v", err)
	}

	stored, _, _ := LoadToken(context.Background(), settings)
	if stored.RefreshToken != "new-rt" {
		t.Errorf("stored refresh token = %q, want new-rt (rotation should persist)", stored.RefreshToken)
	}
}
