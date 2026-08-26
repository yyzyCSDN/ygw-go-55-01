package discover

import (
	"time"

	"scrapehub/internal/model"
)

// RefreshResult describes the outcome of one discovery refresh.
type RefreshResult struct {
	Targets    []model.Target
	Generation int
	Changed    bool
	Err        error
	RefreshedAt time.Time
}

// Refresher periodically pulls the target list from a Provider and tracks
// how many times the list has been reloaded.
type Refresher struct {
	provider   Provider
	generation int
}

// NewRefresher wraps a provider and starts at generation zero.
func NewRefresher(provider Provider) *Refresher {
	return &Refresher{provider: provider}
}

// Refresh reloads the provider and bumps the generation counter.
func (r *Refresher) Refresh() RefreshResult {
	targets, err := r.provider.Refresh()
	result := RefreshResult{Targets: targets, RefreshedAt: time.Now()}
	if err != nil {
		result.Err = err
		return result
	}
	r.generation++
	result.Generation = r.generation
	result.Changed = true
	return result
}

// Current returns the last known list and its generation without re-pulling.
func (r *Refresher) Current() ([]model.Target, int) {
	targets := r.provider.List()
	return targets, r.generation
}

func sortTargets(targets []model.Target) []model.Target {
	for i := 1; i < len(targets); i++ {
		for j := i; j > 0 && targets[j].ID < targets[j-1].ID; j-- {
			targets[j], targets[j-1] = targets[j-1], targets[j]
		}
	}
	return targets
}
