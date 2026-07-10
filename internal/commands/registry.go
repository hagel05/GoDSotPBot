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
type Handler func(cmd slack.SlashCommand, l *league.League) slack.Response

// Registry dispatches slash commands to their handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry builds the registry with all supported commands wired up.
func NewRegistry() *Registry {
	r := &Registry{handlers: make(map[string]Handler)}
	r.register("/draftorder", draftOrder)
	r.register("/draftinfo", draftInfo)
	r.register("/punishment", punishment)
	r.register("/champion", champion)
	r.register("/remotedraft", remoteDraft)
	return r
}

// register normalizes the command name and stores its handler.
func (r *Registry) register(name string, h Handler) {
	r.handlers[strings.ToLower(name)] = h
}

// Dispatch routes a command to its handler. Unknown commands return an ephemeral
// message listing what is available.
func (r *Registry) Dispatch(cmd slack.SlashCommand, l *league.League) slack.Response {
	if h, ok := r.handlers[strings.ToLower(cmd.Command)]; ok {
		return h(cmd, l)
	}
	return slack.Ephemeral(fmt.Sprintf(
		"Sorry, %q isn't a command I know. Try one of: %s",
		cmd.Command, r.commandList(),
	))
}

// commandList returns the supported commands as a sorted, comma-separated string.
func (r *Registry) commandList() string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
