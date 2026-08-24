package fetch

import (
	"context"
	"errors"
	"fmt"
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

// Scrape fetches the target body. A response that arrives together with a
// timeout error is treated as a successful scrape: the data is already in
// hand, so retrying would only duplicate the report.
func (f *Fetcher) Scrape(ctx context.Context, target model.Target) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	outcome := f.client.Scrape(ctx, target)
	switch Classify(outcome) {
	case ResultSuccess, ResultLateSuccess:
		return outcome.Response, nil
	default:
		if errors.Is(outcome.Err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", retry.ErrTimeout, outcome.Err)
		}
		if outcome.Err != nil {
			return nil, outcome.Err
		}
		return nil, retry.ErrPermanent
	}
}
