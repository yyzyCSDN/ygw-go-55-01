package health

import (
	"testing"
	"time"

	"scrapehub/internal/model"
)

func TestJudgeHealthyOnSuccess(t *testing.T) {
	judge := NewJudge(JudgeConfig{ConsecutiveFailures: 3})
	state := judge.Record("node-a", true, time.Now())
	if state != model.HealthHealthy {
		t.Fatalf("expected healthy after success, got %s", state)
	}
	if !judge.Snapshot().Allow("node-a") {
		t.Fatal("healthy target must be allowed")
	}
}

func TestJudgeUnknownAllowsScrape(t *testing.T) {
	judge := NewJudge(JudgeConfig{ConsecutiveFailures: 3})
	if !judge.Allow("fresh-target") {
		t.Fatal("unknown target should still be scraped")
	}
}

func TestJudgeBlocksAfterFailure(t *testing.T) {
	judge := NewJudge(JudgeConfig{ConsecutiveFailures: 3})
	judge.Record("node-a", true, time.Now())
	judge.Record("node-a", false, time.Now())
	if judge.Snapshot().Allow("node-a") {
		t.Fatal("target must not be scraped after a failure")
	}
}
