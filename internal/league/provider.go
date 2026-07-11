package league

import "context"

// Provider is the extensibility seam for league data. Command handlers depend on
// this interface, not on any concrete source, so a Yahoo- or Sleeper-backed
// implementation can be dropped in later without touching the handlers.
type Provider interface {
	// Load returns the current league data. Implementations may cache and may
	// return the same pointer across calls; callers must treat it as read-only.
	Load(ctx context.Context) (*League, error)
}
