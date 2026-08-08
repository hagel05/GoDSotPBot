// Package draft computes the league's draft order from last season's final
// standings, per the league rules.
//
// Rules (10-team league, from final finish order 1 = champion .. N = last):
//   - pick 1 goes to the consolation-bracket winner (7th place)
//   - pick 2 goes to 8th place
//   - picks 3..N go to everyone else in reverse finish order (worst first)
//
// For N = 10 this yields pick -> finish [7, 8, 10, 9, 6, 5, 4, 3, 2, 1]
// (the champion picks last).
package draft

import (
	"errors"
	"fmt"
)

// The two consolation slots that jump to the front of the draft. Named so the
// rule is easy to adjust if the bracket/league size ever changes.
const (
	firstConsolationRank  = 7 // consolation-bracket winner: first overall pick
	secondConsolationRank = 8 // consolation runner-up: second pick
)

// Finish is one team's final placement for the season.
type Finish struct {
	Rank   int    // 1 = champion .. N = last
	Player string // league short name
}

// Order returns the draft order (player names, pick 1 first) computed from the
// season's final standings. It requires a contiguous set of ranks 1..N with no
// duplicates and a non-empty player per rank.
func Order(finishes []Finish) ([]string, error) {
	n := len(finishes)
	if n == 0 {
		return nil, errors.New("no finishers provided")
	}
	if n < secondConsolationRank {
		return nil, fmt.Errorf("need at least %d teams for the consolation rules, got %d", secondConsolationRank, n)
	}

	byRank := make(map[int]string, n)
	for _, finish := range finishes {
		if finish.Rank < 1 || finish.Rank > n {
			return nil, fmt.Errorf("rank %d out of range 1..%d", finish.Rank, n)
		}
		if _, dup := byRank[finish.Rank]; dup {
			return nil, fmt.Errorf("duplicate rank %d", finish.Rank)
		}
		if finish.Player == "" {
			return nil, fmt.Errorf("empty player name for rank %d", finish.Rank)
		}
		byRank[finish.Rank] = finish.Player
	}
	// Range + duplicate checks above guarantee ranks are exactly 1..N when the
	// map has N entries.

	pickRanks := make([]int, 0, n)
	pickRanks = append(pickRanks, firstConsolationRank, secondConsolationRank)
	for rank := n; rank >= 1; rank-- { // remaining teams, worst finish first
		if rank == firstConsolationRank || rank == secondConsolationRank {
			continue
		}
		pickRanks = append(pickRanks, rank)
	}

	order := make([]string, 0, n)
	for _, rank := range pickRanks {
		order = append(order, byRank[rank])
	}
	return order, nil
}
