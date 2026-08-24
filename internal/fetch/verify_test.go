package fetch

import (
	"context"
	"testing"
	"time"

	"scrapehub/internal/forward"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/parse"
	"scrapehub/internal/retry"
)

type verifyLateClient struct {
	calls int
}

func (c *verifyLateClient) Scrape(_ context.Context, _ model.Target) Outcome {
	c.calls++
	if c.calls == 1 {
		return Outcome{
			Response: &Response{Body: []byte("cpu_usage 1.0 1000\n"), Status: 200},
			Err:      retry.ErrTimeout,
		}
	}
	return Outcome{
		Response: &Response{Body: []byte("cpu_usage 1.0 1000\n"), Status: 200},
		Err:      nil,
	}
}

type verifySink struct {
	samples int
}

func (s *verifySink) Send(_ context.Context, batch []model.MetricSample) error {
	s.samples += len(batch)
	return nil
}

func TestFetchTimeoutKeepsReturnedResult(t *testing.T) {
	client := &verifyLateClient{}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(3, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, 50*time.Millisecond, policy, registry)
	sink := &verifySink{}
	forwarder := forward.NewForwarder(forward.NewBuffer(16), sink, 16, registry)
	worker := NewWorker(fetcher, parse.NewParser(), forwarder, policy, registry, 2)

	if err := worker.Run(context.Background(), model.Target{ID: "slow", Endpoint: "http://slow/metrics"}, 1); err != nil {
		t.Fatalf("worker run failed: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("late-returned result must not trigger a retry scrape, got %d calls", client.calls)
	}
	if sink.samples != 1 {
		t.Fatalf("expected the returned data to be forwarded once, got %d samples", sink.samples)
	}
}
