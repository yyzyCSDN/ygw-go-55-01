package schedule

import (
	"sort"
	"time"

	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

// CycleReport summarizes one dispatch pass.
type CycleReport struct {
	Cycle       int64
	Generation  int
	Slots       int
	Planned     int
	Dispatched  []string
	Skipped     []string
	Failed      []string
	Removed     []string
	HealthAt    int64
	Duration    time.Duration
}

// buildPlan flattens stagger windows into one dispatch list. Duplicate IDs
// are dropped so a target can never be dispatched twice in the same cycle.
func buildPlan(windows [][]model.Target) []model.Target {
	seen := make(map[string]struct{})
	plan := make([]model.Target, 0, len(windows))
	for _, window := range windows {
		for _, target := range window {
			if _, exists := seen[target.ID]; exists {
				continue
			}
			seen[target.ID] = struct{}{}
			plan = append(plan, target)
		}
	}
	return plan
}

// spreadTargets reorders a dispatch plan by hash slot so targets from
// different stagger windows interleave instead of hammering the network in
// one contiguous burst.
func spreadTargets(plan []model.Target, slots int) []model.Target {
	out := make([]model.Target, len(plan))
	copy(out, plan)
	sort.SliceStable(out, func(i, j int) bool {
		left := metric.TargetSlot(out[i].ID, slots)
		right := metric.TargetSlot(out[j].ID, slots)
		if left != right {
			return left < right
		}
		return out[i].ID < out[j].ID
	})
	return out
}
