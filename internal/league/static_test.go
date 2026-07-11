package league

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleYAML = `
season: "2025-2026"
draft:
  date: "Sunday, August 31st"
  time: "2:00 PM"
  location: "Hagel House"
  remote_url: "https://sleeper.com/draft/nfl/123"
draft_order: [Jared, Hagel, Dan]
punishments:
  - "First place names last place team"
champion: "Collin"
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestStaticProvider_Load(t *testing.T) {
	path := filepath.Join(t.TempDir(), "league.yaml")
	writeFile(t, path, sampleYAML)

	provider, err := NewStaticProvider(path)
	if err != nil {
		t.Fatalf("NewStaticProvider: %v", err)
	}

	loaded, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Season != "2025-2026" {
		t.Errorf("Season = %q, want 2025-2026", loaded.Season)
	}
	if got, want := len(loaded.DraftOrder), 3; got != want {
		t.Errorf("len(DraftOrder) = %d, want %d", got, want)
	}
	if loaded.Draft.Location != "Hagel House" {
		t.Errorf("Location = %q, want Hagel House", loaded.Draft.Location)
	}
	if loaded.Champion != "Collin" {
		t.Errorf("Champion = %q, want Collin", loaded.Champion)
	}
}

func TestStaticProvider_ReloadsOnChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "league.yaml")
	writeFile(t, path, sampleYAML)

	provider, err := NewStaticProvider(path)
	if err != nil {
		t.Fatalf("NewStaticProvider: %v", err)
	}

	// Rewrite with a new champion and a bumped mtime so the reload triggers.
	writeFile(t, path, `season: "2026-2027"
champion: "Hagel"
draft_order: [Hagel]
`)
	// Ensure the modification time is strictly newer even on coarse-grained FS.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	loaded, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load after change: %v", err)
	}
	if loaded.Champion != "Hagel" {
		t.Errorf("Champion = %q, want Hagel after reload", loaded.Champion)
	}
	if loaded.Season != "2026-2027" {
		t.Errorf("Season = %q, want 2026-2027 after reload", loaded.Season)
	}
}

func TestStaticProvider_KeepsLastGoodOnParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "league.yaml")
	writeFile(t, path, sampleYAML)

	provider, err := NewStaticProvider(path)
	if err != nil {
		t.Fatalf("NewStaticProvider: %v", err)
	}

	// Corrupt the file with invalid YAML and bump mtime.
	writeFile(t, path, "season: \"x\"\n  bad: [unterminated")
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	loaded, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load should keep last-good copy, got error: %v", err)
	}
	if loaded.Season != "2025-2026" {
		t.Errorf("Season = %q, want last-good 2025-2026", loaded.Season)
	}
}

func TestNewStaticProvider_MissingFile(t *testing.T) {
	if _, err := NewStaticProvider(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
