package fetch

import (
	"context"
	"testing"
	"time"

	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/retry"
)

type verifyDeadlineClient struct {
	hasDeadline bool
}

func (c *verifyDeadlineClient) Scrape(ctx context.Context, _ model.Target) Outcome {
	_, c.hasDeadline = ctx.Deadline()
	return Outcome{
		Response: &Response{Body: []byte("cpu_usage 1.0 1000\n"), Status: 200},
		Err:      nil,
	}
}

func TestFetchHonorsTimeoutContext(t *testing.T) {
	client := &verifyDeadlineClient{}
	registry := metric.NewRegistry()
	policy := retry.NewPolicy(2, time.Millisecond, time.Millisecond, registry)
	fetcher := NewFetcher(client, 50*time.Millisecond, policy, registry)
	if _, err := fetcher.Scrape(context.Background(), model.Target{ID: "slow", Endpoint: "http://slow/metrics"}); err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if !client.hasDeadline {
		t.Fatal("scrape request was not given a timeout context")
	}
}
