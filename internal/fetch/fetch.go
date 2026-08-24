package fetch

import (
	"context"
	"time"

	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/retry"
)

// Client performs the raw scrape and reports whether data arrived even when
// the deadline was hit.
type Client interface {
	Scrape(ctx context.Context, target model.Target) Outcome
}

// Outcome carries both the response and the transport error of one scrape.
type Outcome struct {
	Response *Response
	Err      error
}

// Fetcher applies the timeout policy on top of the transport client and
// decides whether a late response should still be honoured.
type Fetcher struct {
	client   Client
	timeout  time.Duration
	policy   *retry.Policy
	registry *metric.Registry
}

// NewFetcher wires a fetcher to its transport and timeout.
func NewFetcher(client Client, timeout time.Duration, policy *retry.Policy, registry *metric.Registry) *Fetcher {
	return &Fetcher{client: client, timeout: timeout, policy: policy, registry: registry}
}

// Scrape fetches the target body. When a response arrives together with a
// timeout error the data is already in hand, so the scrape is treated as a
// success: the response is honoured instead of being discarded and retried,
// which would otherwise report the same batch twice.
func (f *Fetcher) Scrape(ctx context.Context, target model.Target) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	outcome := f.client.Scrape(ctx, target)
	switch Classify(outcome) {
	case ResultLateSuccess:
		// The deadline fired but the body arrived anyway (often a few dozen
		// milliseconds late). Confirm the data is present before honouring it;
		// an empty body is not worth keeping.
		if outcome.Response != nil && len(outcome.Response.Body) > 0 {
			if f.registry != nil {
				f.registry.Inc(metric.MetricScrapeLateSuccess)
			}
			return outcome.Response, nil
		}
		return nil, outcome.Err
	case ResultSuccess:
		return outcome.Response, nil
	default:
		return nil, outcome.Err
	}
}
