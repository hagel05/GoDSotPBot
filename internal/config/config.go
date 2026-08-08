// Package config loads runtime configuration from environment variables.
//
// Nothing install-specific or secret lives in code: every value that can change
// from one deployment to the next is read from the environment here. League
// *content* (draft order, dates, etc.) is deliberately not in this struct — it
// lives in a mounted YAML file described by LeagueConfigPath.
package config

import (
	"fmt"
	"os"
	"strings"
)

// DefaultDBPath is the SQLite path used when DB_PATH is unset. Exported so CLI
// subcommands can reuse it without re-deriving the default.
const DefaultDBPath = "/data/dsotp.db"

// Config holds all runtime configuration for the bot.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// LeagueConfigPath is the path to the mounted league YAML file.
	LeagueConfigPath string

	// LogLevel controls log verbosity (debug, info, warn, error).
	LogLevel string

	// DBPath is the path to the SQLite database file (mounted volume).
	DBPath string

	// AdminUserIDs is the set of Slack user IDs allowed to run guarded commands
	// (e.g. recording a champion).
	AdminUserIDs map[string]bool

	// SlackSigningSecret is used to verify that requests genuinely originate
	// from Slack (HMAC of the raw request body). Required.
	SlackSigningSecret string

	// SlackBotToken is optional today. It is reserved for future features that
	// post to Slack unprompted (e.g. scheduled standings via chat.postMessage).
	SlackBotToken string
}

// Load reads configuration from the environment, applies defaults, and
// validates required values. It returns an error listing every problem found
// so misconfiguration surfaces in one shot at startup rather than at runtime.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		LeagueConfigPath:   getEnv("LEAGUE_CONFIG_PATH", "/config/league.yaml"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DBPath:             getEnv("DB_PATH", DefaultDBPath),
		AdminUserIDs:       parseSet(os.Getenv("ADMIN_USER_IDS")),
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
	}

	var missing []string
	if cfg.SlackSigningSecret == "" {
		missing = append(missing, "SLACK_SIGNING_SECRET")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// getEnv returns the value of key, or fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseSet turns a comma-separated string into a set, ignoring blanks/whitespace.
func parseSet(csv string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range strings.Split(csv, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}
