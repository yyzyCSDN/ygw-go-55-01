package schedule

import (
	"testing"

	"scrapehub/internal/model"
)

func TestStaggerWindowsNoOverlap(t *testing.T) {
	targets := make([]model.Target, 8)
	for i := range targets {
		targets[i] = model.Target{ID: "target-" + string(rune('a'+i)), Endpoint: "http://x/metrics"}
	}
	stagger := NewStagger(4)
	windows := stagger.Split(targets, 4)
	if len(windows) != 4 {
		t.Fatalf("expected 4 windows, got %d", len(windows))
	}
	seen := make(map[string]int)
	for _, window := range windows {
		for _, target := range window {
			seen[target.ID]++
		}
	}
	for id, count := range seen {
		if count > 1 {
			t.Fatalf("target %s appears in %d windows in one cycle", id, count)
		}
	}
}
