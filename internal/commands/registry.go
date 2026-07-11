// Package commands maps Slack slash commands to handlers that render responses
// from league data.
//
// Adding a new command is a two-line change: write a Handler and register it in
// NewRegistry. No control-flow edits, no touching the HTTP layer.
package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

// Handler renders a response for a slash command using the current league data.
type Handler func(command slack.SlashCommand, leagueData *league.League) slack.Response

// Registry dispatches slash commands to their handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry builds the registry with all supported commands wired up.
func NewRegistry() *Registry {
	registry := &Registry{handlers: make(map[string]Handler)}
	registry.register("/draftorder", draftOrder)
	registry.register("/draftinfo", draftInfo)
	registry.register("/punishment", punishment)
	registry.register("/champion", champion)
	registry.register("/remotedraft", remoteDraft)
	return registry
}

// register normalizes the command name and stores its handler.
func (registry *Registry) register(name string, handler Handler) {
	registry.handlers[strings.ToLower(name)] = handler
}

// Dispatch routes a command to its handler. Unknown commands return an ephemeral
// message listing what is available.
func (registry *Registry) Dispatch(command slack.SlashCommand, leagueData *league.League) slack.Response {
	if handler, ok := registry.handlers[strings.ToLower(command.Command)]; ok {
		return handler(command, leagueData)
	}
	return slack.Ephemeral(fmt.Sprintf(
		"Sorry, %q isn't a command I know. Try one of: %s",
		command.Command, registry.commandList(),
	))
}

// commandList returns the supported commands as a sorted, comma-separated string.
func (registry *Registry) commandList() string {
	names := make([]string, 0, len(registry.handlers))
	for name := range registry.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
