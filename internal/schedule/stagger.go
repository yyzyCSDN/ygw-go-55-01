package schedule

import "scrapehub/internal/model"

// Stagger splits a target list into sequential windows so each cycle spreads
// its scrape load evenly instead of hammering every target at once.
type Stagger struct {
	slots int
}

// NewStagger creates a stagger with the given number of windows.
func NewStagger(slots int) *Stagger {
	if slots < 1 {
		slots = 1
	}
	return &Stagger{slots: slots}
}

// Split partitions targets into `slots` windows preserving their order. Each
// window contains a contiguous slice of the sorted list.
func (s *Stagger) Split(targets []model.Target, slots int) [][]model.Target {
	if slots < 1 {
		slots = 1
	}
	windows := make([][]model.Target, slots)
	if len(targets) == 0 {
		return windows
	}
	per := (len(targets) + slots - 1) / slots
	for i, target := range targets {
		index := windowIndex(i, per, slots)
		windows[index] = append(windows[index], target)
		if i%per == per-1 && index+1 < slots {
			windows[index+1] = append(windows[index+1], target)
		}
	}
	return windows
}

// Windows returns the configured slot count.
func (s *Stagger) Windows() int {
	return s.slots
}
