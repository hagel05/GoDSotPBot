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

// Config holds all runtime configuration for the bot.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// LeagueConfigPath is the path to the mounted league YAML file.
	LeagueConfigPath string

	// LogLevel controls log verbosity (debug, info, warn, error).
	LogLevel string

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
