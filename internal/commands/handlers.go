package commands

import (
	"fmt"
	"strings"

	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

// draftOrder lists the draft order as a numbered list.
func draftOrder(_ slack.SlashCommand, leagueData *league.League) slack.Response {
	if len(leagueData.DraftOrder) == 0 {
		return slack.Ephemeral("The draft order hasn't been set yet.")
	}
	return slack.InChannel(numberedList(leagueData.DraftOrder))
}

// draftInfo reports when and where the draft is.
func draftInfo(_ slack.SlashCommand, leagueData *league.League) slack.Response {
	draft := leagueData.Draft
	if draft.Date == "" && draft.Time == "" && draft.Location == "" {
		return slack.Ephemeral("Draft details haven't been set yet.")
	}

	message := "Draft is"
	if draft.Date != "" {
		message += " " + draft.Date
	}
	if draft.Time != "" {
		message += " at " + draft.Time
	}
	if draft.Location != "" {
		message += " - " + draft.Location
	}
	return slack.InChannel(message + ".")
}

// punishment lists the league punishments.
func punishment(_ slack.SlashCommand, leagueData *league.League) slack.Response {
	if len(leagueData.Punishments) == 0 {
		return slack.Ephemeral("No punishments have been set.")
	}
	return slack.InChannel(numberedList(leagueData.Punishments))
}

// champion names the reigning champion.
func champion(_ slack.SlashCommand, leagueData *league.League) slack.Response {
	if leagueData.Champion == "" {
		return slack.Ephemeral("No champion has been recorded yet.")
	}
	return slack.InChannel(fmt.Sprintf("%s is your champion", leagueData.Champion))
}

// remoteDraft shares the remote-draft link.
func remoteDraft(_ slack.SlashCommand, leagueData *league.League) slack.Response {
	if leagueData.Draft.RemoteURL == "" {
		return slack.Ephemeral("No remote draft link has been set.")
	}
	return slack.InChannel(fmt.Sprintf(
		"If unable to attend in person please go here: %s", leagueData.Draft.RemoteURL,
	))
}

// numberedList renders items as "1. a\n2. b\n...".
func numberedList(items []string) string {
	var builder strings.Builder
	for index, item := range items {
		fmt.Fprintf(&builder, "%d. %s", index+1, item)
		if index < len(items)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
