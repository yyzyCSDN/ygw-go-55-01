package fetch

import (
	"context"
	"errors"

	"scrapehub/internal/model"
)

// ErrWorkerBusy is returned when every worker slot is occupied.
var ErrWorkerBusy = errors.New("all scrape workers busy")

// Pool bounds the number of concurrent scrapes.
type Pool struct {
	worker *Worker
	sem    chan struct{}
}

// NewPool creates a pool with the given concurrency.
func NewPool(worker *Worker, size int) *Pool {
	if size < 1 {
		size = 1
	}
	return &Pool{worker: worker, sem: make(chan struct{}, size)}
}

// Run executes a scrape when a slot is free, otherwise reports backpressure.
func (p *Pool) Run(ctx context.Context, target model.Target, cycle int64) error {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
		return p.worker.Run(ctx, target, cycle)
	default:
		return ErrWorkerBusy
	}
}

// Size returns the configured concurrency.
func (p *Pool) Size() int {
	return cap(p.sem)
}
