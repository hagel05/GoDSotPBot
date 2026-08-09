package yahoo

import "testing"

func TestResolveURL(t *testing.T) {
	cases := map[string]string{
		"league/nfl.l.123/standings":  APIBase + "league/nfl.l.123/standings?format=json",
		"/league/nfl.l.123/standings": APIBase + "league/nfl.l.123/standings?format=json",
		"league/nfl.l.123/scoreboard;week=3": APIBase + "league/nfl.l.123/scoreboard;week=3?format=json",
		"https://example.com/x?foo=1": "https://example.com/x?foo=1&format=json",
		"https://example.com/x?format=xml": "https://example.com/x?format=xml",
	}
	for in, want := range cases {
		if got := resolveURL(in); got != want {
			t.Errorf("resolveURL(%q) = %q, want %q", in, got, want)
		}
	}
}
