package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRecordAndByYear(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if err := db.Record(ctx, 2025, "Collin"); err != nil {
		t.Fatalf("Record: %v", err)
	}

	champ, ok, err := db.ByYear(ctx, 2025)
	if err != nil || !ok {
		t.Fatalf("ByYear(2025): ok=%v err=%v", ok, err)
	}
	if champ.Player != "Collin" {
		t.Errorf("player = %q, want Collin", champ.Player)
	}

	if _, ok, _ := db.ByYear(ctx, 1999); ok {
		t.Error("ByYear(1999) should not be found")
	}
}

func TestRecordOverwrites(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	_ = db.Record(ctx, 2025, "Collin")
	if err := db.Record(ctx, 2025, "Hagel"); err != nil {
		t.Fatalf("re-Record: %v", err)
	}

	champ, _, _ := db.ByYear(ctx, 2025)
	if champ.Player != "Hagel" {
		t.Errorf("player = %q, want Hagel after overwrite", champ.Player)
	}
}

func TestRecordEmptyPlayerRejected(t *testing.T) {
	db := newTestDB(t)
	if err := db.Record(context.Background(), 2025, ""); err == nil {
		t.Error("expected error recording empty player")
	}
}

func TestAllAndLatest(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if _, ok, _ := db.Latest(ctx); ok {
		t.Error("Latest on empty DB should not be found")
	}

	_ = db.Record(ctx, 2023, "Dan")
	_ = db.Record(ctx, 2025, "Collin")
	_ = db.Record(ctx, 2024, "Hagel")

	all, err := db.All(ctx)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	wantYears := []int{2025, 2024, 2023} // newest first
	var gotYears []int
	for _, c := range all {
		gotYears = append(gotYears, c.Year)
	}
	if !reflect.DeepEqual(gotYears, wantYears) {
		t.Errorf("All years = %v, want %v", gotYears, wantYears)
	}

	latest, ok, _ := db.Latest(ctx)
	if !ok || latest.Year != 2025 || latest.Player != "Collin" {
		t.Errorf("Latest = %+v, want {2025 Collin}", latest)
	}
}

func TestCountsByPlayer(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	_ = db.Record(ctx, 2020, "Hagel")
	_ = db.Record(ctx, 2021, "Collin")
	_ = db.Record(ctx, 2022, "Hagel")
	_ = db.Record(ctx, 2023, "Hagel")
	_ = db.Record(ctx, 2024, "Collin")

	counts, err := db.CountsByPlayer(ctx)
	if err != nil {
		t.Fatalf("CountsByPlayer: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("got %d players, want 2", len(counts))
	}
	// Hagel (3) should sort before Collin (2).
	if counts[0].Player != "Hagel" || counts[0].Count() != 3 {
		t.Errorf("first = %+v, want Hagel with 3", counts[0])
	}
	if !reflect.DeepEqual(counts[0].Years, []int{2020, 2022, 2023}) {
		t.Errorf("Hagel years = %v, want [2020 2022 2023]", counts[0].Years)
	}
	if counts[1].Player != "Collin" || counts[1].Count() != 2 {
		t.Errorf("second = %+v, want Collin with 2", counts[1])
	}
}

func TestTitlesFor(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	_ = db.Record(ctx, 2020, "Hagel")
	_ = db.Record(ctx, 2023, "Hagel")

	// Case-insensitive lookup, preserving stored capitalization.
	titles, err := db.TitlesFor(ctx, "hagel")
	if err != nil {
		t.Fatalf("TitlesFor: %v", err)
	}
	if titles.Count() != 2 {
		t.Errorf("count = %d, want 2", titles.Count())
	}
	if titles.Player != "Hagel" {
		t.Errorf("player = %q, want Hagel (stored capitalization)", titles.Player)
	}
	if !reflect.DeepEqual(titles.Years, []int{2020, 2023}) {
		t.Errorf("years = %v, want [2020 2023]", titles.Years)
	}

	// A player with no titles.
	none, err := db.TitlesFor(ctx, "Nobody")
	if err != nil {
		t.Fatalf("TitlesFor(Nobody): %v", err)
	}
	if none.Count() != 0 {
		t.Errorf("Nobody count = %d, want 0", none.Count())
	}
}
