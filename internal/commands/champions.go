package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hagel05/go-dsotp-bot/internal/slack"
	"github.com/hagel05/go-dsotp-bot/internal/store"
)

// championPhrase renders "<Team> managed by <Player>", degrading to just the
// player when the team name is unknown (e.g. a hand-entered fallback year).
func championPhrase(teamName, player string) string {
	if teamName != "" {
		return fmt.Sprintf("%s managed by %s", teamName, player)
	}
	return player
}

// champion answers "/champion [year]": no argument reports the reigning
// champion; a year reports that season's champion.
func champion(ctx context.Context, command slack.SlashCommand, env *Env) slack.Response {
	arg := strings.TrimSpace(command.Text)

	if arg == "" {
		latest, ok, err := env.Champions.Latest(ctx)
		if err != nil {
			return storeError(err)
		}
		if !ok {
			return slack.Ephemeral("No champions have been recorded yet.")
		}
		return slack.InChannel(fmt.Sprintf("%s is your champion.",
			championPhrase(latest.TeamName, latest.Player)))
	}

	year, err := strconv.Atoi(arg)
	if err != nil {
		return slack.Ephemeral("Usage: `/champion [year]` — e.g. `/champion 2024`.")
	}
	champ, ok, err := env.Champions.ByYear(ctx, year)
	if err != nil {
		return storeError(err)
	}
	if !ok {
		return slack.Ephemeral(fmt.Sprintf("No champion recorded for %d.", year))
	}
	return slack.InChannel(fmt.Sprintf("The %d champion was %s.",
		champ.Year, championPhrase(champ.TeamName, champ.Player)))
}

// champions lists the full championship history, newest first.
func champions(ctx context.Context, _ slack.SlashCommand, env *Env) slack.Response {
	all, err := env.Champions.All(ctx)
	if err != nil {
		return storeError(err)
	}
	if len(all) == 0 {
		return slack.Ephemeral("No champions have been recorded yet.")
	}

	var builder strings.Builder
	builder.WriteString("Championship history:\n")
	for _, champ := range all {
		if champ.TeamName != "" {
			fmt.Fprintf(&builder, "%d — %s (%s)\n", champ.Year, champ.Player, champ.TeamName)
		} else {
			fmt.Fprintf(&builder, "%d — %s\n", champ.Year, champ.Player)
		}
	}
	return slack.InChannel(strings.TrimRight(builder.String(), "\n"))
}

// titles answers "/titles [player]": a player reports their count and years; no
// argument reports the all-time leaderboard.
func titles(ctx context.Context, command slack.SlashCommand, env *Env) slack.Response {
	arg := strings.TrimSpace(command.Text)

	if arg == "" {
		leaderboard, err := env.Champions.CountsByPlayer(ctx)
		if err != nil {
			return storeError(err)
		}
		if len(leaderboard) == 0 {
			return slack.Ephemeral("No champions have been recorded yet.")
		}
		var builder strings.Builder
		builder.WriteString("Championships won:\n")
		for rank, entry := range leaderboard {
			fmt.Fprintf(&builder, "%d. %s — %d (%s)\n",
				rank+1, entry.Player, entry.Count(), joinYears(entry.Years))
		}
		return slack.InChannel(strings.TrimRight(builder.String(), "\n"))
	}

	record, err := env.Champions.TitlesFor(ctx, arg)
	if err != nil {
		return storeError(err)
	}
	if record.Count() == 0 {
		return slack.InChannel(fmt.Sprintf("%s has no championships… yet.", arg))
	}
	return slack.InChannel(fmt.Sprintf("%s has %d championship(s): %s.",
		record.Player, record.Count(), joinYears(record.Years)))
}

// recordChampion records (or corrects) a season's champion. Guarded: only admins
// listed in ADMIN_USER_IDS may run it. The player is a single token; anything
// after it is the (optional) team name.
func recordChampion(ctx context.Context, command slack.SlashCommand, env *Env) slack.Response {
	if !env.IsAdmin(command.UserID) {
		return slack.Ephemeral("Sorry, only league admins can record champions.")
	}

	fields := strings.Fields(command.Text)
	if len(fields) < 2 {
		return slack.Ephemeral("Usage: `/recordchampion <year> <player> [team name]` — e.g. `/recordchampion 2024 Hagel Vamonos Pest`.")
	}
	year, err := strconv.Atoi(fields[0])
	if err != nil {
		return slack.Ephemeral("The first argument must be a year, e.g. `/recordchampion 2024 Hagel Vamonos Pest`.")
	}
	player := fields[1]
	teamName := strings.Join(fields[2:], " ")

	if err := env.Champions.Record(ctx, store.Champion{Year: year, TeamName: teamName, Player: player}); err != nil {
		return storeError(err)
	}
	return slack.InChannel(fmt.Sprintf("Recorded %s as the %d champion.",
		championPhrase(teamName, player), year))
}

// joinYears formats a list of years as "2020, 2022, 2023".
func joinYears(years []int) string {
	parts := make([]string, len(years))
	for i, year := range years {
		parts[i] = strconv.Itoa(year)
	}
	return strings.Join(parts, ", ")
}

// storeError returns a generic ephemeral message; the real error is logged by
// the caller/handler layer.
func storeError(_ error) slack.Response {
	return slack.Ephemeral("Sorry, I couldn't reach the championship records right now.")
}
