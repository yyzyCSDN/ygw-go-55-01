package fetch

import (
	"context"
	"fmt"

	"scrapehub/internal/forward"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/parse"
	"scrapehub/internal/retry"
)

// Worker runs one scrape through fetch, parse and forward.
type Worker struct {
	fetcher   *Fetcher
	parser    *parse.Parser
	forwarder *forward.Forwarder
	policy    *retry.Policy
	registry  *metric.Registry
	shards    int
}

// NewWorker wires a worker to its pipeline stages.
func NewWorker(
	fetcher *Fetcher,
	parser *parse.Parser,
	forwarder *forward.Forwarder,
	policy *retry.Policy,
	registry *metric.Registry,
	shards int,
) *Worker {
	if shards < 1 {
		shards = 1
	}
	return &Worker{fetcher: fetcher, parser: parser, forwarder: forwarder, policy: policy, registry: registry, shards: shards}
}

// Run scrapes one target, parses the body and forwards the samples.
func (w *Worker) Run(ctx context.Context, target model.Target, cycle int64) error {
	shard := w.shardFor(target)
	w.registry.Set(fmt.Sprintf("%s_shard_%d", metric.MetricWorkersBusy, shard), 1)
	defer w.registry.Set(fmt.Sprintf("%s_shard_%d", metric.MetricWorkersBusy, shard), 0)
	w.registry.Set(metric.MetricConnectionsOpen, w.registry.Get(metric.MetricConnectionsOpen)+1)
	defer w.registry.Set(metric.MetricConnectionsOpen, w.registry.Get(metric.MetricConnectionsOpen)-1)

	task := model.NewTask(target.ID, cycle)
	_ = task.Transition(model.TaskScheduled)
	_ = task.Transition(model.TaskFetching)
	state := retry.NewAttemptState()

	var response *Response
	err := w.policy.Execute(ctx, func(attempt int) error {
		var scrapeErr error
		response, scrapeErr = w.fetcher.Scrape(ctx, target)
		state.Record(scrapeErr, w.policy.ShouldRetry(scrapeErr, attempt+1))
		if scrapeErr != nil {
			w.registry.Inc(metric.MetricScrapeFailures)
		}
		return scrapeErr
	})
	if err != nil {
		task.Attempt = state.Attempt
		task.RecordError(err.Error())
		w.recordTaskState(task)
		return err
	}
	w.registry.Inc(metric.MetricScrapes)
	_ = task.Transition(model.TaskParsed)

	samples, err := w.parser.ParseBody(response.Body, target.ID)
	if err != nil {
		task.RecordError(err.Error())
		w.recordTaskState(task)
		return fmt.Errorf("parse %s: %w", target.ID, err)
	}
	samples = dedupeSamples(samples)
	_ = task.Transition(model.TaskForwarded)
	w.recordTaskState(task)
	if err := w.forwarder.Forward(ctx, samples); err != nil {
		task.RecordError(err.Error())
		w.recordTaskState(task)
		return err
	}
	return w.forwarder.Flush(ctx)
}

// shardFor maps a target to one of the configured worker shards.
func (w *Worker) shardFor(target model.Target) int {
	return metric.TargetSlot(target.ID, w.shards)
}

// recordTaskState publishes the current task state into the metrics registry.
func (w *Worker) recordTaskState(task model.Task) {
	w.registry.Set(metric.MetricTaskState, float64(task.State))
}

// dedupeSamples drops samples that appear more than once in one response.
func dedupeSamples(samples []model.MetricSample) []model.MetricSample {
	seen := make(map[string]struct{}, len(samples))
	out := samples[:0]
	for _, sample := range samples {
		key := sample.Key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, sample)
	}
	return out
}
