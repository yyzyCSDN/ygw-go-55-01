package health

import (
	"testing"
	"time"

	"scrapehub/internal/model"
)

func TestHealthRemovalRequiresConsecutiveFailures(t *testing.T) {
	judge := NewJudge(JudgeConfig{ConsecutiveFailures: 3})
	judge.Record("node-a", true, time.Now())
	state := judge.Record("node-a", false, time.Now())
	if state == model.HealthRemoved {
		t.Fatal("a single failure must not remove the target from the scrape list")
	}
	judge.Record("node-a", true, time.Now())
	if judge.State("node-a") != model.HealthHealthy {
		t.Fatalf("recovered target should be healthy, got %s", judge.State("node-a"))
	}
}
