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

	mu       sync.RWMutex
	cached   *League
	modTime  time.Time
	loadedAt time.Time
}

// NewStaticProvider constructs a provider backed by the file at path and loads
// it once so configuration errors surface at startup.
func NewStaticProvider(path string) (*StaticProvider, error) {
	p := &StaticProvider{path: path}
	if _, err := p.reload(); err != nil {
		return nil, fmt.Errorf("loading league file %q: %w", path, err)
	}
	return p, nil
}

// Load returns the current league data, reloading from disk first if the file
// has changed since the last read.
func (p *StaticProvider) Load(_ context.Context) (*League, error) {
	info, err := os.Stat(p.path)
	if err != nil {
		// If we already have a cached copy, keep serving it rather than failing
		// a command because of a transient stat error.
		if l := p.current(); l != nil {
			slog.Warn("stat league file failed; serving cached copy", "path", p.path, "error", err)
			return l, nil
		}
		return nil, fmt.Errorf("stat league file %q: %w", p.path, err)
	}

	if info.ModTime().Equal(p.lastModTime()) {
		return p.current(), nil
	}

	l, err := p.reload()
	if err != nil {
		if cached := p.current(); cached != nil {
			slog.Error("reloading league file failed; keeping previous copy", "path", p.path, "error", err)
			return cached, nil
		}
		return nil, err
	}
	slog.Info("reloaded league file", "path", p.path, "season", l.Season)
	return l, nil
}

// reload reads and parses the file, updating the cache on success.
func (p *StaticProvider) reload() (*League, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("read league file: %w", err)
	}

	var l League
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse league yaml: %w", err)
	}

	info, err := os.Stat(p.path)
	if err != nil {
		return nil, fmt.Errorf("stat league file: %w", err)
	}

	p.mu.Lock()
	p.cached = &l
	p.modTime = info.ModTime()
	p.loadedAt = time.Now()
	p.mu.Unlock()

	return &l, nil
}

func (p *StaticProvider) current() *League {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cached
}

func (p *StaticProvider) lastModTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.modTime
}
