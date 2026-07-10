package commands

import (
	"fmt"
	"strings"

	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

// draftOrder lists the draft order as a numbered list.
func draftOrder(_ slack.SlashCommand, l *league.League) slack.Response {
	if len(l.DraftOrder) == 0 {
		return slack.Ephemeral("The draft order hasn't been set yet.")
	}
	return slack.InChannel(numberedList(l.DraftOrder))
}

// draftInfo reports when and where the draft is.
func draftInfo(_ slack.SlashCommand, l *league.League) slack.Response {
	d := l.Draft
	if d.Date == "" && d.Time == "" && d.Location == "" {
		return slack.Ephemeral("Draft details haven't been set yet.")
	}

	msg := "Draft is"
	if d.Date != "" {
		msg += " " + d.Date
	}
	if d.Time != "" {
		msg += " at " + d.Time
	}
	if d.Location != "" {
		msg += " - " + d.Location
	}
	return slack.InChannel(msg + ".")
}

// punishment lists the league punishments.
func punishment(_ slack.SlashCommand, l *league.League) slack.Response {
	if len(l.Punishments) == 0 {
		return slack.Ephemeral("No punishments have been set.")
	}
	return slack.InChannel(numberedList(l.Punishments))
}

// champion names the reigning champion.
func champion(_ slack.SlashCommand, l *league.League) slack.Response {
	if l.Champion == "" {
		return slack.Ephemeral("No champion has been recorded yet.")
	}
	return slack.InChannel(fmt.Sprintf("%s is your champion", l.Champion))
}

// remoteDraft shares the remote-draft link.
func remoteDraft(_ slack.SlashCommand, l *league.League) slack.Response {
	if l.Draft.RemoteURL == "" {
		return slack.Ephemeral("No remote draft link has been set.")
	}
	return slack.InChannel(fmt.Sprintf(
		"If unable to attend in person please go here: %s", l.Draft.RemoteURL,
	))
}

// numberedList renders items as "1. a\n2. b\n...".
func numberedList(items []string) string {
	var b strings.Builder
	for i, item := range items {
		fmt.Fprintf(&b, "%d. %s", i+1, item)
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
