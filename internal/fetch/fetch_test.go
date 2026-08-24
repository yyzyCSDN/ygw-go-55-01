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
