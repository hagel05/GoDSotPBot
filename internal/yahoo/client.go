package yahoo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// APIBase is the Yahoo Fantasy Sports API v2 base URL.
const APIBase = "https://fantasysports.yahooapis.com/fantasy/v2/"

// NewHTTPClient returns an *http.Client that authenticates every request with
// the stored (auto-refreshing, self-persisting) Yahoo token. It errors if no
// token is stored yet — authorize first via the browser flow or `bot yahoo-auth`.
func NewHTTPClient(ctx context.Context, cfg *oauth2.Config, settings SettingsStore) (*http.Client, error) {
	src, err := TokenSource(ctx, cfg, settings)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, src), nil
}

// Fetch performs an authenticated GET against a Yahoo Fantasy API resource. The
// argument may be a path relative to APIBase (e.g. "league/nfl.l.12345/standings")
// or an absolute URL; `format=json` is appended if absent. It returns the HTTP
// status and the raw response body.
func Fetch(ctx context.Context, httpClient *http.Client, pathOrURL string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolveURL(pathOrURL), nil)
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, body, nil
}

// resolveURL turns a path or absolute URL into a full JSON API URL.
func resolveURL(pathOrURL string) string {
	url := pathOrURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = APIBase + strings.TrimPrefix(url, "/")
	}
	switch {
	case !strings.Contains(url, "?"):
		url += "?format=json"
	case !strings.Contains(url, "format="):
		url += "&format=json"
	}
	return url
}
