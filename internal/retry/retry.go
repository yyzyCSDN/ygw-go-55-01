package retry

import (
	"context"
	"errors"
	"time"

	"scrapehub/internal/metric"
)

// ErrTimeout is the sentinel raised when a scrape deadline is exceeded.
var ErrTimeout = errors.New("scrape deadline exceeded")

// ErrPermanent marks failures that must never be retried.
var ErrPermanent = errors.New("permanent failure")

// Policy decides whether and how a failed scrape is retried.
type Policy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Registry    *metric.Registry
}

// NewPolicy builds a retry policy with sane defaults.
func NewPolicy(maxAttempts int, baseDelay, maxDelay time.Duration, registry *metric.Registry) *Policy {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &Policy{MaxAttempts: maxAttempts, BaseDelay: baseDelay, MaxDelay: maxDelay, Registry: registry}
}

// ShouldRetry reports whether an error is worth another attempt.
func (p *Policy) ShouldRetry(err error, attempt int) bool {
	if err == nil || attempt >= p.MaxAttempts {
		return false
	}
	if errors.Is(err, ErrPermanent) {
		return false
	}
	return true
}

// Backoff returns how long to wait before the next attempt.
func (p *Policy) Backoff(attempt int) time.Duration {
	return ComputeBackoff(p.BaseDelay, attempt, p.MaxDelay)
}

// Execute runs fn until it succeeds, the policy is exhausted, or ctx ends.
func (p *Policy) Execute(ctx context.Context, fn func(attempt int) error) error {
	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn(attempt)
		if lastErr == nil {
			return nil
		}
		if p.Registry != nil {
			p.Registry.Inc(metric.MetricRetries)
		}
		if !p.ShouldRetry(lastErr, attempt+1) {
			break
		}
		delay := p.Backoff(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}
