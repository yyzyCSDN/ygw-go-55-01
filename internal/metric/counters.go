package metric

const (
	// MetricDispatched counts targets handed to the dispatcher.
	MetricDispatched = "scrape_dispatched_total"
	// MetricSkipped counts targets filtered out by health gating.
	MetricSkipped = "scrape_skipped_total"
	// MetricScrapes counts successful fetch executions.
	MetricScrapes = "scrape_ok_total"
	// MetricScrapeFailures counts failed fetch executions.
	MetricScrapeFailures = "scrape_failed_total"
	// MetricForwarded counts samples written to the downstream sink.
	MetricForwarded = "forwarded_samples_total"
	// MetricForwardDropped counts samples dropped by backpressure.
	MetricForwardDropped = "forwarded_dropped_total"
	// MetricConnectionsOpen is a gauge of in-flight scrape connections.
	MetricConnectionsOpen = "scrape_connections_open"
	// MetricWorkersBusy is a gauge of occupied fetch workers.
	MetricWorkersBusy = "scrape_workers_busy"
	// MetricRetries counts retry attempts performed by the policy.
	MetricRetries = "retry_attempts_total"
	// MetricTaskState tracks the most recent task state machine value.
	MetricTaskState = "task_state"
)
