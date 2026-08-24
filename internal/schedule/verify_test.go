package schedule

import (
	"context"
	"testing"
	"time"

	"scrapehub/internal/discover"
	"scrapehub/internal/health"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

type verifyDispatcher struct {
	ids []string
}

func (d *verifyDispatcher) Dispatch(_ context.Context, target model.Target, _ int64) error {
	d.ids = append(d.ids, target.ID)
	return nil
}

func TestScheduleStopsRemovedTarget(t *testing.T) {
	provider := discover.NewStaticProvider(
		model.Target{ID: "node-a", Endpoint: "http://a/metrics"},
		model.Target{ID: "node-b", Endpoint: "http://b/metrics"},
	)
	store := discover.NewListStore(discover.NewRefresher(provider))
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 2})
	registry := metric.NewRegistry()
	dispatcher := &verifyDispatcher{}
	scheduler := NewScheduler(store, NewStagger(3), dispatcher, judge, registry, time.Second, 3, NewJobHistory(8))

	scheduler.RunCycle(time.Now())
	provider.Remove("node-a")
	scheduler.RunCycle(time.Now().Add(time.Minute))

	counts := make(map[string]int)
	for _, id := range dispatcher.ids {
		counts[id]++
	}
	if counts["node-a"] != 1 {
		t.Fatalf("removed target kept being scraped: %v", counts)
	}
	if counts["node-b"] != 2 {
		t.Fatalf("remaining target should be scraped every cycle: %v", counts)
	}
}
