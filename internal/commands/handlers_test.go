package commands

import (
	"strings"
	"testing"

	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

func testLeague() *league.League {
	return &league.League{
		Season: "2025-2026",
		Draft: league.Draft{
			Date:      "Sunday, August 31st",
			Time:      "2:00 PM",
			Location:  "Hagel House",
			RemoteURL: "https://sleeper.com/draft/nfl/123",
		},
		DraftOrder:  []string{"Jared", "Hagel", "Dan"},
		Punishments: []string{"Name the last place team", "Buy the drinks"},
		Champion:    "Collin",
	}
}

func dispatch(t *testing.T, command string, l *league.League) slack.Response {
	t.Helper()
	return NewRegistry().Dispatch(slack.SlashCommand{Command: command}, l)
}

func TestDraftOrder(t *testing.T) {
	resp := dispatch(t, "/draftorder", testLeague())
	if resp.ResponseType != slack.ResponseInChannel {
		t.Errorf("response type = %q, want in_channel", resp.ResponseType)
	}
	want := "1. Jared\n2. Hagel\n3. Dan"
	if resp.Text != want {
		t.Errorf("text = %q, want %q", resp.Text, want)
	}
}

func TestDraftInfo(t *testing.T) {
	resp := dispatch(t, "/draftinfo", testLeague())
	want := "Draft is Sunday, August 31st at 2:00 PM - Hagel House."
	if resp.Text != want {
		t.Errorf("text = %q, want %q", resp.Text, want)
	}
}

func TestPunishment(t *testing.T) {
	resp := dispatch(t, "/punishment", testLeague())
	if !strings.Contains(resp.Text, "1. Name the last place team") {
		t.Errorf("text missing first punishment: %q", resp.Text)
	}
}

func TestChampion(t *testing.T) {
	resp := dispatch(t, "/champion", testLeague())
	if resp.Text != "Collin is your champion" {
		t.Errorf("text = %q", resp.Text)
	}
}

func TestRemoteDraft(t *testing.T) {
	resp := dispatch(t, "/remotedraft", testLeague())
	if !strings.Contains(resp.Text, "https://sleeper.com/draft/nfl/123") {
		t.Errorf("text missing remote url: %q", resp.Text)
	}
}

func TestUnknownCommand(t *testing.T) {
	resp := dispatch(t, "/bogus", testLeague())
	if resp.ResponseType != slack.ResponseEphemeral {
		t.Errorf("unknown command should be ephemeral, got %q", resp.ResponseType)
	}
	if !strings.Contains(resp.Text, "/draftinfo") {
		t.Errorf("unknown command help should list commands: %q", resp.Text)
	}
}

func TestMissingDataIsEphemeral(t *testing.T) {
	empty := &league.League{}
	for _, cmd := range []string{"/draftorder", "/draftinfo", "/punishment", "/champion", "/remotedraft"} {
		resp := dispatch(t, cmd, empty)
		if resp.ResponseType != slack.ResponseEphemeral {
			t.Errorf("%s with no data: response type = %q, want ephemeral", cmd, resp.ResponseType)
		}
	}
}

func TestCommandCaseInsensitive(t *testing.T) {
	resp := dispatch(t, "/DraftOrder", testLeague())
	if resp.ResponseType != slack.ResponseInChannel {
		t.Errorf("mixed-case command should dispatch, got %q: %q", resp.ResponseType, resp.Text)
	}
}
