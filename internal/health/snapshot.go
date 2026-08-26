package health

import "scrapehub/internal/model"

// Snapshot is a consistent read of the health judge at one moment.
type Snapshot struct {
	states map[string]model.HealthState
}

// buildSnapshot copies the judge state into an immutable snapshot.
func buildSnapshot(states map[string]model.HealthState) Snapshot {
	copy := make(map[string]model.HealthState, len(states))
	for id, state := range states {
		copy[id] = state
	}
	return Snapshot{states: copy}
}

// Allow reports whether the target should still be scraped.
func (s Snapshot) Allow(id string) bool {
	return s.State(id) != model.HealthRemoved && s.State(id) != model.HealthUnhealthy
}

// State returns the health state of a target in this snapshot.
func (s Snapshot) State(id string) model.HealthState {
	return s.states[id]
}
