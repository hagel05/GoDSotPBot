package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func testAuthServer(user, pass string) (*AuthServer, *fakeSettings) {
	settings := newFakeSettings()
	cfg := OAuthConfig("client-id", "secret", "https://host/yahoo/callback")
	return NewAuthServer(cfg, settings, user, pass), settings
}

func TestLogin_NoBasicAuth_RedirectsToYahoo(t *testing.T) {
	auth, _ := testAuthServer("", "") // no app gate
	rec := httptest.NewRecorder()
	auth.LoginHandler(rec, httptest.NewRequest(http.MethodGet, "/yahoo/login", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, authURL) {
		t.Errorf("redirect = %q, want Yahoo auth URL", loc)
	}
	if !strings.Contains(loc, "state=") {
		t.Errorf("redirect missing state: %q", loc)
	}
}

func TestLogin_BasicAuthGate(t *testing.T) {
	auth, _ := testAuthServer("me", "s3cret")

	// No credentials -> 401 with a challenge.
	rec := httptest.NewRecorder()
	auth.LoginHandler(rec, httptest.NewRequest(http.MethodGet, "/yahoo/login", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no creds: status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("WWW-Authenticate"), "Basic") {
		t.Error("missing WWW-Authenticate challenge")
	}

	// Wrong credentials -> 401.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/yahoo/login", nil)
	req.SetBasicAuth("me", "wrong")
	auth.LoginHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong creds: status = %d, want 401", rec.Code)
	}

	// Correct credentials -> 302 to Yahoo.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/yahoo/login", nil)
	req.SetBasicAuth("me", "s3cret")
	auth.LoginHandler(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("correct creds: status = %d, want 302", rec.Code)
	}
}

func TestCallback_InvalidState(t *testing.T) {
	auth, _ := testAuthServer("", "")
	rec := httptest.NewRecorder()
	auth.CallbackHandler(rec, httptest.NewRequest(http.MethodGet, "/yahoo/callback?state=bogus&code=abc", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad state", rec.Code)
	}
}

func TestCallback_HappyPath(t *testing.T) {
	auth, settings := testAuthServer("", "")
	// Stub the token exchange so we don't hit Yahoo.
	auth.exchange = func(_ context.Context, code string) (*oauth2.Token, error) {
		if code != "good-code" {
			t.Errorf("exchange got code %q", code)
		}
		return &oauth2.Token{RefreshToken: "rt-web", AccessToken: "at"}, nil
	}

	state, err := auth.states.issue()
	if err != nil {
		t.Fatalf("issue state: %v", err)
	}

	rec := httptest.NewRecorder()
	auth.CallbackHandler(rec, httptest.NewRequest(http.MethodGet, "/yahoo/callback?state="+state+"&code=good-code", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Yahoo connected") {
		t.Errorf("missing success page: %q", rec.Body.String())
	}

	stored, ok, _ := LoadToken(context.Background(), settings)
	if !ok || stored.RefreshToken != "rt-web" {
		t.Errorf("token not saved: ok=%v tok=%+v", ok, stored)
	}
}

func TestState_SingleUseAndExpiry(t *testing.T) {
	store := newStateStore()

	state, _ := store.issue()
	if !store.consume(state) {
		t.Error("first consume should succeed")
	}
	if store.consume(state) {
		t.Error("second consume should fail (single-use)")
	}

	// Manually insert an already-expired state.
	store.issued["expired"] = time.Now().Add(-time.Minute)
	if store.consume("expired") {
		t.Error("expired state should not validate")
	}
}
