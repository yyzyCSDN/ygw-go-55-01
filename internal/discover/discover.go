package discover

import (
	"time"

	"scrapehub/internal/model"
)

// ListStore hands out live target snapshots and walks the list with a cursor
// so each scrape cycle consumes every target exactly once.
type ListStore struct {
	refresher  *Refresher
	cursor     int
	dispatched map[string]int64
}

// NewListStore builds a store backed by the given refresher.
func NewListStore(refresher *Refresher) *ListStore {
	return &ListStore{
		refresher:  refresher,
		dispatched: make(map[string]int64),
	}
}

// Snapshot returns the live list and its generation.
func (s *ListStore) Snapshot() Snapshot {
	targets, generation := s.refresher.Current()
	return Snapshot{Targets: targets, Generation: generation, CreatedAt: time.Now().UnixMilli()}
}

// Refresh reloads the provider and resets the walk cursor so a changed list
// starts from its first target in the following cycle.
func (s *ListStore) Refresh() Snapshot {
	result := s.refresher.Refresh()
	s.cursor = 0
	s.dispatched = make(map[string]int64)
	return Snapshot{Targets: result.Targets, Generation: result.Generation, CreatedAt: result.RefreshedAt.UnixMilli()}
}

// Next returns the next target that has not been dispatched in the cycle.
// When the cursor reaches the end it wraps around so short lists still get a
// full walk before any target repeats.
func (s *ListStore) Next(cycle int64) (model.Target, bool) {
	targets, _ := s.refresher.Current()
	count := len(targets)
	if count == 0 {
		return model.Target{}, false
	}
	for examined := 0; examined < count; examined++ {
		position := s.cursor % count
		s.cursor++
		target := targets[position]
		if last, seen := s.dispatched[target.ID]; seen && last == cycle {
			continue
		}
		s.dispatched[target.ID] = cycle
		return target, true
	}
	return model.Target{}, false
}

// Len returns the number of targets currently visible.
func (s *ListStore) Len() int {
	targets, _ := s.refresher.Current()
	return len(targets)
}

// Remove deletes a target from the underlying provider.
func (s *ListStore) Remove(id string) {
	if provider, ok := s.refresher.provider.(*StaticProvider); ok {
		provider.Remove(id)
	}
}
