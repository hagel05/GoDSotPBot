package draft

import (
	"reflect"
	"strconv"
	"testing"
)

// finishOrder builds a []Finish where each rank's player is "P<rank>".
func finishOrder(n int) []Finish {
	finishes := make([]Finish, 0, n)
	for rank := 1; rank <= n; rank++ {
		finishes = append(finishes, Finish{Rank: rank, Player: "P" + strconv.Itoa(rank)})
	}
	return finishes
}

func TestOrder_TenTeams(t *testing.T) {
	order, err := Order(finishOrder(10))
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	// pick -> finish [7, 8, 10, 9, 6, 5, 4, 3, 2, 1]
	want := []string{"P7", "P8", "P10", "P9", "P6", "P5", "P4", "P3", "P2", "P1"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v\nwant  %v", order, want)
	}
}

func TestOrder_EightTeams(t *testing.T) {
	order, err := Order(finishOrder(8))
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	// [7, 8, then 6,5,4,3,2,1]
	want := []string{"P7", "P8", "P6", "P5", "P4", "P3", "P2", "P1"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v\nwant  %v", order, want)
	}
}

func TestOrder_ChampionPicksLast(t *testing.T) {
	order, _ := Order(finishOrder(10))
	if order[len(order)-1] != "P1" {
		t.Errorf("last pick = %q, want the champion P1", order[len(order)-1])
	}
	if order[0] != "P7" {
		t.Errorf("first pick = %q, want 7th place P7", order[0])
	}
}

func TestOrder_Errors(t *testing.T) {
	cases := map[string][]Finish{
		"empty":          {},
		"too few teams":  finishOrder(6),
		"duplicate rank": {{1, "a"}, {1, "b"}, {3, "c"}, {4, "d"}, {5, "e"}, {6, "f"}, {7, "g"}, {8, "h"}},
		"rank out of range": {
			{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}, {5, "e"}, {6, "f"}, {7, "g"}, {99, "h"},
		},
		"empty player": {
			{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}, {5, "e"}, {6, "f"}, {7, "g"}, {8, ""},
		},
	}
	for name, finishes := range cases {
		if _, err := Order(finishes); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}
