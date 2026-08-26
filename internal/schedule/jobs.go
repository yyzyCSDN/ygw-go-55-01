package schedule

import (
	"sync"
	"time"

	"scrapehub/internal/model"
)

// JobRecord is one target's task entry inside a cycle.
type JobRecord struct {
	TargetID string
	Cycle    int64
	State    model.TaskState
	Attempt  int
	Err      string
	At       int64
}

// JobHistory keeps a bounded ring of the most recent task records so the
// monitor can show what the scheduler has been doing.
type JobHistory struct {
	mu    sync.Mutex
	items []JobRecord
	max   int
}

// NewJobHistory creates a history that retains up to max records.
func NewJobHistory(max int) *JobHistory {
	if max < 1 {
		max = 1
	}
	return &JobHistory{max: max}
}

// Add appends a record, evicting the oldest when the ring is full.
func (h *JobHistory) Add(record JobRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items = append(h.items, record)
	if len(h.items) > h.max {
		h.items = h.items[len(h.items)-h.max:]
	}
}

// Recent returns the stored records newest first.
func (h *JobHistory) Recent() []JobRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]JobRecord, len(h.items))
	for i := range h.items {
		out[len(h.items)-1-i] = h.items[i]
	}
	return out
}

// RecordDispatch stores a successful dispatch for one target.
func (h *JobHistory) RecordDispatch(targetID string, cycle int64, attempt int) {
	h.Add(JobRecord{
		TargetID: targetID,
		Cycle:    cycle,
		State:    model.TaskForwarded,
		Attempt:  attempt,
		At:       time.Now().UnixMilli(),
	})
}

// RecordFailure stores a failed dispatch for one target.
func (h *JobHistory) RecordFailure(targetID string, cycle int64, attempt int, message string) {
	h.Add(JobRecord{
		TargetID: targetID,
		Cycle:    cycle,
		State:    model.TaskFailed,
		Attempt:  attempt,
		Err:      message,
		At:       time.Now().UnixMilli(),
	})
}
