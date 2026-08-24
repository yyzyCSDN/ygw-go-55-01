package fetch

import (
	"context"
	"strings"
	"testing"
	"time"

	"scrapehub/internal/forward"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/parse"
	"scrapehub/internal/retry"
)

type fakeClient struct {
	outcome Outcome
}

func (f fakeClient) Scrape(_ context.Context, _ model.Target) Outcome {
	return f.outcome
}

// countingClient reports each scrape attempt and serves the configured outcome
// on the first call. It is used to assert that a late success is not retried.
type countingClient struct {
	calls   int
	outcome Outcome
}

func (c *countingClient) Scrape(_ context.Context, _ model.Target) Outcome {
	c.calls++
	return c.outcome
}

func TestFetcherSuccess(t *testing.T) {
	client := fakeClient{outcome: Outcome{
		Response: &Response{Body: []byte("cpu 1.0 1000"), Status: 200},
	}}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(2, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, time.Second, policy, registry)
	response, err := fetcher.Scrape(context.Background(), model.Target{ID: "a", Endpoint: "http://a"})
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if !strings.Contains(string(response.Body), "cpu") {
		t.Fatalf("unexpected body: %q", response.Body)
	}
}

type recordingSink struct {
	batches int
	samples int
}

func (s *recordingSink) Send(_ context.Context, samples []model.MetricSample) error {
	s.batches++
	s.samples += len(samples)
	return nil
}

func TestWorkerForwardsParsedSamples(t *testing.T) {
	client := fakeClient{outcome: Outcome{
		Response: &Response{Body: []byte("cpu 1.0 1000\nmem 2.0 1000\n"), Status: 200},
	}}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(2, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, time.Second, policy, registry)
	sink := &recordingSink{}
	forwarder := forward.NewForwarder(forward.NewBuffer(16), sink, 16, registry)
	worker := NewWorker(fetcher, parse.NewParser(), forwarder, policy, registry, 2)
	if err := worker.Run(context.Background(), model.Target{ID: "a", Endpoint: "http://a"}, 1); err != nil {
		t.Fatalf("worker run failed: %v", err)
	}
	if sink.samples != 2 {
		t.Fatalf("expected 2 forwarded samples, got %d", sink.samples)
	}
}

// TestFetcherHonoursLateSuccess reproduces the duplicate-report bug: a slow
// target returns the body together with a scrape timeout error. The fetcher
// must honour the data instead of dropping it and letting the worker retry,
// which would otherwise report the same batch twice.
func TestFetcherHonoursLateSuccess(t *testing.T) {
	client := &countingClient{outcome: Outcome{
		Response: &Response{Body: []byte("cpu 1.0 1000\nmem 2.0 1000\n"), Status: 200},
		Err:      retry.ErrTimeout,
	}}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(3, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, time.Second, policy, registry)

	response, err := fetcher.Scrape(context.Background(), model.Target{ID: "a", Endpoint: "http://a"})
	if err != nil {
		t.Fatalf("late success must not surface an error: %v", err)
	}
	if response == nil || len(response.Body) == 0 {
		t.Fatalf("late response must be honoured, got %v", response)
	}
	if client.calls != 1 {
		t.Fatalf("late success must not be retried, got %d scrape calls", client.calls)
	}
}

// TestWorkerDoesNotDuplicateLateSuccess runs the late-success outcome through
// the whole worker and asserts the samples are forwarded exactly once.
func TestWorkerDoesNotDuplicateLateSuccess(t *testing.T) {
	client := &countingClient{outcome: Outcome{
		Response: &Response{Body: []byte("cpu 1.0 1000\nmem 2.0 1000\n"), Status: 200},
		Err:      retry.ErrTimeout,
	}}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(3, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, time.Second, policy, registry)
	sink := &recordingSink{}
	forwarder := forward.NewForwarder(forward.NewBuffer(16), sink, 16, registry)
	worker := NewWorker(fetcher, parse.NewParser(), forwarder, policy, registry, 2)

	if err := worker.Run(context.Background(), model.Target{ID: "a", Endpoint: "http://a"}, 1); err != nil {
		t.Fatalf("worker run failed: %v", err)
	}
	if sink.batches != 1 || sink.samples != 2 {
		t.Fatalf("expected a single batch of 2 samples, got batches=%d samples=%d", sink.batches, sink.samples)
	}
	if client.calls != 1 {
		t.Fatalf("late success must not trigger a retry scrape, got %d calls", client.calls)
	}
}

// TestFetcherDropsEmptyLateSuccess ensures an empty late response is not
// honoured: no body means nothing to report, so the error is surfaced for a
// normal retry.
func TestFetcherDropsEmptyLateSuccess(t *testing.T) {
	client := &countingClient{outcome: Outcome{
		Response: &Response{Body: nil, Status: 200},
		Err:      retry.ErrTimeout,
	}}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(3, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, time.Second, policy, registry)

	if _, err := fetcher.Scrape(context.Background(), model.Target{ID: "a", Endpoint: "http://a"}); err == nil {
		t.Fatal("empty late response must surface its error")
	}
}
