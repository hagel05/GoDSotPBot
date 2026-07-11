package league

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// StaticProvider serves league data from a YAML file on disk.
//
// It reloads automatically when the file's modification time changes, so edits
// made on TrueNAS take effect without restarting the container. If a reload
// fails to parse, the last known-good value is kept and the error is logged.
type StaticProvider struct {
	path string

	mu      sync.RWMutex
	cached  *League
	modTime time.Time
}

// NewStaticProvider constructs a provider backed by the file at path and loads
// it once so configuration errors surface at startup.
func NewStaticProvider(path string) (*StaticProvider, error) {
	provider := &StaticProvider{path: path}
	if _, err := provider.reload(); err != nil {
		return nil, fmt.Errorf("loading league file %q: %w", path, err)
	}
	return provider, nil
}

// Load returns the current league data, reloading from disk first if the file
// has changed since the last read.
func (provider *StaticProvider) Load(_ context.Context) (*League, error) {
	info, err := os.Stat(provider.path)
	if err != nil {
		// If we already have a cached copy, keep serving it rather than failing
		// a command because of a transient stat error.
		if cached := provider.current(); cached != nil {
			slog.Warn("stat league file failed; serving cached copy", "path", provider.path, "error", err)
			return cached, nil
		}
		return nil, fmt.Errorf("stat league file %q: %w", provider.path, err)
	}

	if info.ModTime().Equal(provider.lastModTime()) {
		return provider.current(), nil
	}

	loaded, err := provider.reload()
	if err != nil {
		if cached := provider.current(); cached != nil {
			slog.Error("reloading league file failed; keeping previous copy", "path", provider.path, "error", err)
			return cached, nil
		}
		return nil, err
	}
	slog.Info("reloaded league file", "path", provider.path, "season", loaded.Season)
	return loaded, nil
}

// reload reads and parses the file, updating the cache on success.
func (provider *StaticProvider) reload() (*League, error) {
	rawBytes, err := os.ReadFile(provider.path)
	if err != nil {
		return nil, fmt.Errorf("read league file: %w", err)
	}

	var parsed League
	if err := yaml.Unmarshal(rawBytes, &parsed); err != nil {
		return nil, fmt.Errorf("parse league yaml: %w", err)
	}

	info, err := os.Stat(provider.path)
	if err != nil {
		return nil, fmt.Errorf("stat league file: %w", err)
	}

	provider.mu.Lock()
	provider.cached = &parsed
	provider.modTime = info.ModTime()
	provider.mu.Unlock()

	return &parsed, nil
}

func (provider *StaticProvider) current() *League {
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	return provider.cached
}

func (provider *StaticProvider) lastModTime() time.Time {
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	return provider.modTime
}
