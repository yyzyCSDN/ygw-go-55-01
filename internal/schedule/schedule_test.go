package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"scrapehub/internal/discover"
	"scrapehub/internal/health"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

type recordingDispatcher struct {
	ids []string
	err error
}

func (d *recordingDispatcher) Dispatch(_ context.Context, target model.Target, _ int64) error {
	if d.err != nil {
		return d.err
	}
	d.ids = append(d.ids, target.ID)
	return nil
}

func TestSchedulerEmptyCycle(t *testing.T) {
	store := discover.NewListStore(discover.NewRefresher(discover.NewStaticProvider()))
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 2})
	registry := metric.NewRegistry()
	dispatcher := &recordingDispatcher{}
	scheduler := NewScheduler(store, NewStagger(3), dispatcher, judge, registry, time.Second, 3, NewJobHistory(8))
	report := scheduler.RunCycle(time.Now())
	if report.Planned != 0 || len(report.Dispatched) != 0 {
		t.Fatalf("empty cycle should plan and dispatch nothing: %+v", report)
	}
}

func TestSchedulerDispatchesTargets(t *testing.T) {
	provider := discover.NewStaticProvider(
		model.Target{ID: "node-a", Endpoint: "http://a/metrics"},
		model.Target{ID: "node-b", Endpoint: "http://b/metrics"},
	)
	store := discover.NewListStore(discover.NewRefresher(provider))
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 2})
	registry := metric.NewRegistry()
	dispatcher := &recordingDispatcher{}
	scheduler := NewScheduler(store, NewStagger(3), dispatcher, judge, registry, time.Second, 3, NewJobHistory(8))
	report := scheduler.RunCycle(time.Now())
	if len(report.Dispatched) != 2 {
		t.Fatalf("expected 2 dispatched targets, got %+v", report.Dispatched)
	}
	if len(dispatcher.ids) != 2 {
		t.Fatalf("dispatcher should see both targets: %+v", dispatcher.ids)
	}
}

func TestStaggerSingleWindowNoOverlap(t *testing.T) {
	targets := make([]model.Target, 6)
	for i := range targets {
		targets[i] = model.Target{ID: string(rune('a' + i)), Endpoint: "http://x/metrics"}
	}
	stagger := NewStagger(1)
	windows := stagger.Split(targets, 1)
	if len(windows) != 1 || len(windows[0]) != 6 {
		t.Fatalf("single window should contain all targets, got %d windows", len(windows))
	}
}

func TestBuildPlanDeduplicates(t *testing.T) {
	target := model.Target{ID: "dup", Endpoint: "http://d/metrics"}
	windows := [][]model.Target{{target}, {target}}
	plan := buildPlan(windows)
	if len(plan) != 1 {
		t.Fatalf("buildPlan must drop duplicate ids, got %d", len(plan))
	}
}

func TestSchedulerRecordsFailedTarget(t *testing.T) {
	provider := discover.NewStaticProvider(model.Target{ID: "bad", Endpoint: "http://b/metrics"})
	store := discover.NewListStore(discover.NewRefresher(provider))
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 2})
	registry := metric.NewRegistry()
	dispatcher := &recordingDispatcher{err: errors.New("scrape failed")}
	scheduler := NewScheduler(store, NewStagger(2), dispatcher, judge, registry, time.Second, 2, NewJobHistory(8))
	report := scheduler.RunCycle(time.Now())
	if len(report.Failed) != 1 {
		t.Fatalf("expected one failed target, got %+v", report)
	}
}
