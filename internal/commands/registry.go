// Package commands maps Slack slash commands to handlers that render responses
// from league data and persistent storage.
//
// Adding a new command is a two-line change: write a Handler and register it in
// NewRegistry. No control-flow edits, no touching the HTTP layer.
package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

// Handler renders a response for a slash command using the request Env.
type Handler func(ctx context.Context, command slack.SlashCommand, env *Env) slack.Response

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
	registry.register("/remotedraft", remoteDraft)
	registry.register("/champion", champion)
	registry.register("/champions", champions)
	registry.register("/titles", titles)
	registry.register("/recordchampion", recordChampion)
	return registry
}

// register normalizes the command name and stores its handler.
func (registry *Registry) register(name string, handler Handler) {
	registry.handlers[strings.ToLower(name)] = handler
}

// Dispatch routes a command to its handler. Unknown commands return an ephemeral
// message listing what is available.
func (registry *Registry) Dispatch(ctx context.Context, command slack.SlashCommand, env *Env) slack.Response {
	if handler, ok := registry.handlers[strings.ToLower(command.Command)]; ok {
		return handler(ctx, command, env)
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
