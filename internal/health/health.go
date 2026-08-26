package health

import (
	"sync"
	"time"

	"scrapehub/internal/model"
)

// JudgeConfig controls how quickly a target is demoted or removed.
type JudgeConfig struct {
	// ConsecutiveFailures is how many failed scrapes in a row move a target
	// from unhealthy to removed.
	ConsecutiveFailures int
}

// Judge tracks the health state machine for every known target.
type Judge struct {
	config      JudgeConfig
	mu          sync.RWMutex
	states      map[string]model.HealthState
	consecutive map[string]int
}

// NewJudge builds a judge with the given configuration.
func NewJudge(config JudgeConfig) *Judge {
	if config.ConsecutiveFailures < 1 {
		config.ConsecutiveFailures = 1
	}
	return &Judge{
		config:      config,
		states:      make(map[string]model.HealthState),
		consecutive: make(map[string]int),
	}
}

// Record feeds one scrape result into the health state machine and returns
// the target's new state.
func (j *Judge) Record(id string, ok bool, now time.Time) model.HealthState {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.states[id] == model.HealthRemoved {
		return model.HealthRemoved
	}
	state := evaluate(j.states[id], j.consecutive[id], ok, j.config.ConsecutiveFailures)
	j.states[id] = state
	if ok {
		j.consecutive[id] = 0
	} else {
		j.consecutive[id]++
	}
	return state
}

// State returns the current health state of a target.
func (j *Judge) State(id string) model.HealthState {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.states[id]
}

// Allow reports whether the target should still be scraped.
func (j *Judge) Allow(id string) bool {
	return j.Snapshot().Allow(id)
}

// Snapshot returns a consistent view of all health states.
func (j *Judge) Snapshot() Snapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return buildSnapshot(j.states)
}
