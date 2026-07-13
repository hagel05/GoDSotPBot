package commands

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hagel05/go-dsotp-bot/internal/slack"
	"github.com/hagel05/go-dsotp-bot/internal/store"
)

func envWithChampions(t *testing.T, admins map[string]bool) *Env {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &Env{Champions: db, Admins: admins}
}

func run(t *testing.T, env *Env, command, text, userID string) slack.Response {
	t.Helper()
	cmd := slack.SlashCommand{Command: command, Text: text, UserID: userID}
	return NewRegistry().Dispatch(context.Background(), cmd, env)
}

func TestChampionEmpty(t *testing.T) {
	env := envWithChampions(t, nil)
	resp := run(t, env, "/champion", "", "")
	if resp.ResponseType != slack.ResponseEphemeral {
		t.Errorf("empty /champion should be ephemeral, got %q", resp.ResponseType)
	}
}

func TestChampionLatestAndByYear(t *testing.T) {
	env := envWithChampions(t, nil)
	ctx := context.Background()
	_ = env.Champions.Record(ctx, 2023, "Dan")
	_ = env.Champions.Record(ctx, 2025, "Collin")

	latest := run(t, env, "/champion", "", "")
	if !strings.Contains(latest.Text, "Collin") || !strings.Contains(latest.Text, "2025") {
		t.Errorf("/champion latest = %q, want Collin/2025", latest.Text)
	}

	byYear := run(t, env, "/champion", "2023", "")
	if !strings.Contains(byYear.Text, "Dan") {
		t.Errorf("/champion 2023 = %q, want Dan", byYear.Text)
	}

	missing := run(t, env, "/champion", "1999", "")
	if missing.ResponseType != slack.ResponseEphemeral {
		t.Errorf("/champion 1999 should be ephemeral, got %q", missing.ResponseType)
	}

	bad := run(t, env, "/champion", "notayear", "")
	if bad.ResponseType != slack.ResponseEphemeral {
		t.Errorf("/champion notayear should be ephemeral usage, got %q", bad.ResponseType)
	}
}

func TestChampionsList(t *testing.T) {
	env := envWithChampions(t, nil)
	ctx := context.Background()
	_ = env.Champions.Record(ctx, 2024, "Hagel")
	_ = env.Champions.Record(ctx, 2025, "Collin")

	resp := run(t, env, "/champions", "", "")
	if !strings.Contains(resp.Text, "2025 — Collin") || !strings.Contains(resp.Text, "2024 — Hagel") {
		t.Errorf("/champions = %q, want both years", resp.Text)
	}
	// Newest first.
	if strings.Index(resp.Text, "2025") > strings.Index(resp.Text, "2024") {
		t.Errorf("/champions not newest-first: %q", resp.Text)
	}
}

func TestTitles(t *testing.T) {
	env := envWithChampions(t, nil)
	ctx := context.Background()
	_ = env.Champions.Record(ctx, 2020, "Hagel")
	_ = env.Champions.Record(ctx, 2023, "Hagel")
	_ = env.Champions.Record(ctx, 2024, "Collin")

	board := run(t, env, "/titles", "", "")
	if !strings.Contains(board.Text, "Hagel — 2") {
		t.Errorf("/titles leaderboard = %q, want Hagel — 2", board.Text)
	}

	player := run(t, env, "/titles", "hagel", "") // case-insensitive
	if !strings.Contains(player.Text, "Hagel has 2 championship") {
		t.Errorf("/titles hagel = %q", player.Text)
	}
	if !strings.Contains(player.Text, "2020, 2023") {
		t.Errorf("/titles hagel missing years: %q", player.Text)
	}

	none := run(t, env, "/titles", "Nobody", "")
	if !strings.Contains(none.Text, "no championships") {
		t.Errorf("/titles Nobody = %q", none.Text)
	}
}

func TestRecordChampionGuard(t *testing.T) {
	env := envWithChampions(t, map[string]bool{"U_ADMIN": true})

	// Non-admin is refused and nothing is written.
	denied := run(t, env, "/recordchampion", "2026 Eric", "U_RANDO")
	if denied.ResponseType != slack.ResponseEphemeral || !strings.Contains(denied.Text, "admin") {
		t.Errorf("non-admin record = %q (%s), want admin refusal", denied.Text, denied.ResponseType)
	}
	if _, ok, _ := env.Champions.ByYear(context.Background(), 2026); ok {
		t.Error("non-admin record should not have written a row")
	}

	// Admin succeeds and the value is queryable.
	ok := run(t, env, "/recordchampion", "2026 Eric", "U_ADMIN")
	if ok.ResponseType != slack.ResponseInChannel || !strings.Contains(ok.Text, "Eric") {
		t.Errorf("admin record = %q (%s), want success", ok.Text, ok.ResponseType)
	}
	got := run(t, env, "/champion", "2026", "")
	if !strings.Contains(got.Text, "Eric") {
		t.Errorf("after record, /champion 2026 = %q, want Eric", got.Text)
	}
}

func TestRecordChampionUsage(t *testing.T) {
	env := envWithChampions(t, map[string]bool{"U_ADMIN": true})
	resp := run(t, env, "/recordchampion", "onlyonearg", "U_ADMIN")
	if resp.ResponseType != slack.ResponseEphemeral {
		t.Errorf("bad usage should be ephemeral, got %q", resp.ResponseType)
	}
}
