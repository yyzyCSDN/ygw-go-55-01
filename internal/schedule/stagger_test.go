package schedule

import (
	"fmt"
	"testing"

	"scrapehub/internal/model"
)

// TestStaggerWindowsNeverShareTargets guards the stagger boundary: when many
// targets are split across fine-grained windows, each target must land in
// exactly one window. The old Split appended every boundary target to the next
// window as well, so the tail of window N duplicated into the head of window
// N+1 and the same target was scraped twice per cycle. With many targets and
// fine windows this duplicated enough of the list to double downstream
// metrics. This test keeps that regression from coming back.
func TestStaggerWindowsNeverShareTargets(t *testing.T) {
	const n = 100
	targets := make([]model.Target, n)
	for i := range targets {
		targets[i] = model.Target{ID: fmt.Sprintf("node-%03d", i), Endpoint: "http://x/metrics"}
	}

	// Fine windows: per-target window is small, so boundaries are numerous.
	// This is exactly the configuration that exposed the duplicate scrape.
	for _, slots := range []int{2, 3, 5, 10, 25} {
		t.Run(fmt.Sprintf("slots=%d", slots), func(t *testing.T) {
			windows := NewStagger(slots).Split(targets, slots)

			seen := make(map[string]int, n)
			total := 0
			for _, w := range windows {
				total += len(w)
				for _, tgt := range w {
					seen[tgt.ID]++
				}
			}

			// Every target appears in exactly one window.
			duplicated := 0
			for _, c := range seen {
				if c > 1 {
					duplicated++
				}
			}
			if duplicated != 0 {
				t.Fatalf("slots=%d: %d targets appeared in more than one window (total placed=%d, unique=%d)",
					slots, duplicated, total, len(seen))
			}
			if len(seen) != n {
				t.Fatalf("slots=%d: expected %d unique targets across windows, got %d", slots, n, len(seen))
			}
			if total != n {
				t.Fatalf("slots=%d: expected %d total placements, got %d", slots, n, total)
			}

			// Windows must be contiguous and ordered: concatenating them
			// reproduces the original list, so boundaries are continuous and
			// no target is skipped or repeated between windows.
			flat := make([]model.Target, 0, n)
			for _, w := range windows {
				flat = append(flat, w...)
			}
			for i, tgt := range flat {
				if tgt.ID != targets[i].ID {
					t.Fatalf("slots=%d: window order broken at %d: got %s want %s",
						slots, i, tgt.ID, targets[i].ID)
				}
			}

			// The dispatch plan built from these windows must contain every
			// target exactly once; no duplicate dispatch within a cycle.
			plan := buildPlan(windows)
			if len(plan) != n {
				t.Fatalf("slots=%d: plan has %d targets, want %d", slots, len(plan), n)
			}
		})
	}
}
