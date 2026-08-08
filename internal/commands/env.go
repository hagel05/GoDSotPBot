package commands

import (
	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/store"
)

// Env carries everything a command handler may need. It is built per request:
// League is the current (hot-reloaded) static content, while Champions and the
// admin set are long-lived singletons wired in at startup.
type Env struct {
	League    *league.League
	Champions store.ChampionStore
	Admins    map[string]bool
}

// IsAdmin reports whether the given Slack user ID is allowed to run guarded
// commands (e.g. recording a champion).
func (env *Env) IsAdmin(userID string) bool {
	return userID != "" && env.Admins[userID]
}
