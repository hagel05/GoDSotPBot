package yahoo

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// stateTTL is how long an issued OAuth state remains valid.
const stateTTL = 10 * time.Minute

// AuthServer serves the browser-based Yahoo OAuth flow: GET /yahoo/login
// redirects to Yahoo, GET /yahoo/callback completes it and stores the token.
//
// The callback is protected by a single-use, expiring state (CSRF). Initiation
// is optionally gated by app-level HTTP Basic Auth; when no credentials are set,
// no app gate is applied (protect the path at the reverse proxy instead).
type AuthServer struct {
	cfg      *oauth2.Config
	settings SettingsStore
	user     string
	pass     string
	states   *stateStore

	// exchange is overridable in tests; defaults to cfg.Exchange.
	exchange func(ctx context.Context, code string) (*oauth2.Token, error)
}

// NewAuthServer builds an AuthServer. basicUser/basicPass enable app-level Basic
// Auth when both are non-empty.
func NewAuthServer(cfg *oauth2.Config, settings SettingsStore, basicUser, basicPass string) *AuthServer {
	auth := &AuthServer{
		cfg:      cfg,
		settings: settings,
		user:     basicUser,
		pass:     basicPass,
		states:   newStateStore(),
	}
	auth.exchange = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return cfg.Exchange(ctx, code)
	}
	return auth
}

// BasicAuthConfigured reports whether the app-level Basic Auth gate is enabled.
func (a *AuthServer) BasicAuthConfigured() bool { return a.user != "" && a.pass != "" }

// checkBasicAuth returns true if the request may proceed. If a gate is
// configured and the request fails it, checkBasicAuth writes a 401 and returns
// false.
func (a *AuthServer) checkBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	if !a.BasicAuthConfigured() {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if ok &&
		subtle.ConstantTimeCompare([]byte(user), []byte(a.user)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(a.pass)) == 1 {
		return true
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="DSotP Yahoo Auth"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}

// LoginHandler gates the request, issues a state, and redirects to Yahoo.
func (a *AuthServer) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if !a.checkBasicAuth(w, r) {
		return
	}
	state, err := a.states.issue()
	if err != nil {
		slog.Error("issue oauth state", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, AuthCodeURL(a.cfg, state), http.StatusFound)
}

// CallbackHandler validates the state, exchanges the code, and persists the token.
func (a *AuthServer) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if !a.checkBasicAuth(w, r) {
		return
	}
	if authErr := r.URL.Query().Get("error"); authErr != "" {
		http.Error(w, "authorization denied: "+authErr, http.StatusBadRequest)
		return
	}
	if !a.states.consume(r.URL.Query().Get("state")) {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := a.exchange(r.Context(), code)
	if err != nil {
		slog.Error("yahoo token exchange failed", "error", err)
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}
	if err := SaveToken(r.Context(), a.settings, token); err != nil {
		slog.Error("save yahoo token", "error", err)
		http.Error(w, "failed to save token", http.StatusInternalServerError)
		return
	}

	slog.Info("yahoo authorization complete; token saved")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, successPage)
}

const successPage = `<!doctype html><html><head><meta charset="utf-8"><title>Connected</title></head>
<body style="font-family:system-ui,sans-serif;max-width:32rem;margin:4rem auto;text-align:center">
<h1>✅ Yahoo connected</h1>
<p>The bot is authorized. You can close this tab.</p>
</body></html>`

// stateStore holds issued OAuth states with an expiry. States are single-use.
type stateStore struct {
	mu     sync.Mutex
	issued map[string]time.Time // state -> expiry
}

func newStateStore() *stateStore {
	return &stateStore{issued: make(map[string]time.Time)}
}

// issue mints a random state and records it with a TTL.
func (s *stateStore) issue() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := hex.EncodeToString(buf)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.issued[state] = time.Now().Add(stateTTL)
	return state, nil
}

// consume validates and removes a state; it returns false if unknown or expired.
func (s *stateStore) consume(state string) bool {
	if state == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.issued[state]
	if !ok {
		return false
	}
	delete(s.issued, state)
	return time.Now().Before(expiry)
}

// gcLocked drops expired states. Caller must hold the lock.
func (s *stateStore) gcLocked() {
	now := time.Now()
	for state, expiry := range s.issued {
		if now.After(expiry) {
			delete(s.issued, state)
		}
	}
}
