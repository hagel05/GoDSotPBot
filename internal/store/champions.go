package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

// Champion is a single season's champion.
type Champion struct {
	Year   int
	Player string
}

// PlayerTitles is a player's championship record.
type PlayerTitles struct {
	Player string
	Years  []int // ascending
}

// Count returns how many championships the player has won.
func (t PlayerTitles) Count() int { return len(t.Years) }

// ChampionStore is the persistence seam the command handlers depend on.
type ChampionStore interface {
	Record(ctx context.Context, year int, player string) error
	ByYear(ctx context.Context, year int) (Champion, bool, error)
	All(ctx context.Context) ([]Champion, error)
	Latest(ctx context.Context) (Champion, bool, error)
	CountsByPlayer(ctx context.Context) ([]PlayerTitles, error)
	TitlesFor(ctx context.Context, player string) (PlayerTitles, error)
}

// Ensure *DB satisfies ChampionStore.
var _ ChampionStore = (*DB)(nil)

// Record inserts or updates the champion for a year (re-recording corrects a
// prior entry).
func (db *DB) Record(ctx context.Context, year int, player string) error {
	if player == "" {
		return errors.New("player must not be empty")
	}
	_, err := db.sql.ExecContext(ctx,
		`INSERT INTO champions (year, player) VALUES (?, ?)
		 ON CONFLICT(year) DO UPDATE SET player = excluded.player`,
		year, player)
	if err != nil {
		return fmt.Errorf("record champion: %w", err)
	}
	return nil
}

// ByYear returns the champion for a year; the bool is false if none is recorded.
func (db *DB) ByYear(ctx context.Context, year int) (Champion, bool, error) {
	var player string
	err := db.sql.QueryRowContext(ctx,
		`SELECT player FROM champions WHERE year = ?`, year).Scan(&player)
	if errors.Is(err, sql.ErrNoRows) {
		return Champion{}, false, nil
	}
	if err != nil {
		return Champion{}, false, fmt.Errorf("champion by year: %w", err)
	}
	return Champion{Year: year, Player: player}, true, nil
}

// All returns every recorded champion, newest year first.
func (db *DB) All(ctx context.Context) ([]Champion, error) {
	rows, err := db.sql.QueryContext(ctx,
		`SELECT year, player FROM champions ORDER BY year DESC`)
	if err != nil {
		return nil, fmt.Errorf("all champions: %w", err)
	}
	defer rows.Close()

	var champions []Champion
	for rows.Next() {
		var c Champion
		if err := rows.Scan(&c.Year, &c.Player); err != nil {
			return nil, fmt.Errorf("scan champion: %w", err)
		}
		champions = append(champions, c)
	}
	return champions, rows.Err()
}

// Latest returns the most recent champion; the bool is false if none exist.
func (db *DB) Latest(ctx context.Context) (Champion, bool, error) {
	var c Champion
	err := db.sql.QueryRowContext(ctx,
		`SELECT year, player FROM champions ORDER BY year DESC LIMIT 1`).Scan(&c.Year, &c.Player)
	if errors.Is(err, sql.ErrNoRows) {
		return Champion{}, false, nil
	}
	if err != nil {
		return Champion{}, false, fmt.Errorf("latest champion: %w", err)
	}
	return c, true, nil
}

// CountsByPlayer returns each player's title record, most titles first (ties
// broken alphabetically).
func (db *DB) CountsByPlayer(ctx context.Context) ([]PlayerTitles, error) {
	champions, err := db.All(ctx)
	if err != nil {
		return nil, err
	}

	byPlayer := make(map[string][]int)
	for _, c := range champions {
		byPlayer[c.Player] = append(byPlayer[c.Player], c.Year)
	}

	titles := make([]PlayerTitles, 0, len(byPlayer))
	for player, years := range byPlayer {
		sort.Ints(years)
		titles = append(titles, PlayerTitles{Player: player, Years: years})
	}
	sort.Slice(titles, func(i, j int) bool {
		if titles[i].Count() != titles[j].Count() {
			return titles[i].Count() > titles[j].Count()
		}
		return titles[i].Player < titles[j].Player
	})
	return titles, nil
}

// TitlesFor returns a single player's record (case-insensitive match). A player
// with no titles yields a zero Count and empty Years.
func (db *DB) TitlesFor(ctx context.Context, player string) (PlayerTitles, error) {
	rows, err := db.sql.QueryContext(ctx,
		`SELECT year, player FROM champions WHERE player = ? COLLATE NOCASE ORDER BY year ASC`,
		player)
	if err != nil {
		return PlayerTitles{}, fmt.Errorf("titles for player: %w", err)
	}
	defer rows.Close()

	result := PlayerTitles{Player: player}
	for rows.Next() {
		var year int
		var name string
		if err := rows.Scan(&year, &name); err != nil {
			return PlayerTitles{}, fmt.Errorf("scan title: %w", err)
		}
		// Preserve the stored capitalization of the name.
		result.Player = name
		result.Years = append(result.Years, year)
	}
	return result, rows.Err()
}
