package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scrapehub/internal/discover"
	"scrapehub/internal/fetch"
	"scrapehub/internal/forward"
	"scrapehub/internal/health"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/parse"
	"scrapehub/internal/retry"
	"scrapehub/internal/schedule"
)

type emptyClient struct{}

func (emptyClient) Scrape(_ context.Context, _ model.Target) fetch.Outcome {
	return fetch.Outcome{Response: &fetch.Response{Body: []byte(""), Status: 200}}
}

func buildTestServer(t *testing.T) *Server {
	t.Helper()
	registry := metric.NewRegistry()
	provider := discover.NewStaticProvider(
		model.Target{ID: "node-a", Endpoint: "http://127.0.0.1:9100/metrics"},
	)
	store := discover.NewListStore(discover.NewRefresher(provider))
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 3})
	policy := retry.NewPolicy(2, time.Millisecond, time.Millisecond, registry)
	fetcher := fetch.NewFetcher(emptyClient{}, time.Second, policy, registry)
	parser := parse.NewParser()
	forwarder := forward.NewForwarder(forward.NewBuffer(16), &discardSink{}, 16, registry)
	worker := fetch.NewWorker(fetcher, parser, forwarder, policy, registry, 2)
	dispatcher := &scrapeDispatcher{pool: fetch.NewPool(worker, 2)}
	history := schedule.NewJobHistory(8)
	scheduler := schedule.NewScheduler(store, schedule.NewStagger(3), dispatcher, judge, registry, time.Minute, 3, history)
	return NewServer(scheduler, store, judge, registry, history, provider)
}

type discardSink struct{}

func (d *discardSink) Send(_ context.Context, _ []model.MetricSample) error {
	return nil
}

func TestHealthz(t *testing.T) {
	server := buildTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	server.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("healthz returned %d", recorder.Code)
	}
}

func TestTargetsEndpoint(t *testing.T) {
	server := buildTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil)
	recorder := httptest.NewRecorder()
	server.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("targets returned %d", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("targets body must not be empty")
	}
}
