package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	var (
		addr        = flag.String("addr", ":8080", "HTTP listen address")
		interval    = flag.Duration("interval", 30*time.Second, "scrape cycle interval")
		slots       = flag.Int("slots", 3, "stagger window count")
		batchSize   = flag.Int("batch-size", 64, "forward batch size")
		timeout     = flag.Duration("timeout", 5*time.Second, "per-scrape timeout")
		maxAttempts = flag.Int("max-attempts", 3, "retry attempts per target")
		downstream  = flag.String("downstream", "http://127.0.0.1:9090/write", "downstream sink endpoint")
		sdEndpoint  = flag.String("sd-endpoint", "", "optional service discovery endpoint")
	)
	flag.Parse()

	registry := metric.NewRegistry()
	var provider discover.Provider
	var staticProvider *discover.StaticProvider
	if *sdEndpoint != "" {
		provider = discover.NewHTTPSDProvider(*sdEndpoint, &http.Client{Timeout: 5 * time.Second})
	} else {
		staticProvider = discover.NewStaticProvider(
			model.Target{ID: "node-a", Name: "node-a", Endpoint: "http://127.0.0.1:9100/metrics", Kind: model.TargetStatic},
			model.Target{ID: "node-b", Name: "node-b", Endpoint: "http://127.0.0.1:9101/metrics", Kind: model.TargetStatic},
		)
		provider = staticProvider
	}
	refresher := discover.NewRefresher(provider)
	store := discover.NewListStore(refresher)
	judge := health.NewJudge(health.JudgeConfig{ConsecutiveFailures: 3})
	policy := retry.NewPolicy(*maxAttempts, 200*time.Millisecond, 2*time.Second, registry)

	transport := fetch.NewHTTPClient(http.DefaultTransport)
	transportClient := wrapClient(transport)
	fetcher := fetch.NewFetcher(transportClient, *timeout, policy, registry)
	parser := parse.NewParser()
	sink := forward.NewHTTPSink(*downstream, 3*time.Second)
	buffer := forward.NewBuffer(1024)
	forwarder := forward.NewForwarder(buffer, sink, *batchSize, registry)
	worker := fetch.NewWorker(fetcher, parser, forwarder, policy, registry, *slots)
	pool := fetch.NewPool(worker, *slots)
	dispatcher := &scrapeDispatcher{pool: pool}

	stagger := schedule.NewStagger(*slots)
	history := schedule.NewJobHistory(256)
	scheduler := schedule.NewScheduler(store, stagger, dispatcher, judge, registry, *interval, *slots, history)

	server := NewServer(scheduler, store, judge, registry, history, staticProvider)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go runCycles(ctx, scheduler, scheduler.Interval())

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("scrapehub listening on %s", *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func runCycles(ctx context.Context, scheduler *schedule.Scheduler, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			report := scheduler.RunCycle(now)
			log.Printf("cycle %d: planned=%d dispatched=%d skipped=%d failed=%d removed=%v",
				report.Cycle, report.Planned, len(report.Dispatched), len(report.Skipped), len(report.Failed), report.Removed)
		}
	}
}
