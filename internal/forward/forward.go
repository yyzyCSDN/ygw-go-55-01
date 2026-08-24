package forward

import (
	"context"
	"errors"

	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

// ErrBackpressure is returned when the forward buffer is full and the sink is
// too slow to keep up.
var ErrBackpressure = errors.New("forward buffer full, applying backpressure")

// Forwarder stages parsed samples in an ordered buffer and flushes them to a
// downstream sink in time-ordered batches.
type Forwarder struct {
	buffer    *Buffer
	sink      Sink
	batchSize int
	registry  *metric.Registry
}

// NewForwarder wires a forwarder to its sink.
func NewForwarder(buffer *Buffer, sink Sink, batchSize int, registry *metric.Registry) *Forwarder {
	if batchSize < 1 {
		batchSize = 1
	}
	return &Forwarder{buffer: buffer, sink: sink, batchSize: batchSize, registry: registry}
}

// Forward accepts samples, appends them to the buffer and flushes when the
// batch threshold is reached.
func (f *Forwarder) Forward(ctx context.Context, samples []model.MetricSample) error {
	if f.buffer.Full() {
		f.registry.Add(metric.MetricForwardDropped, float64(len(samples)))
		return ErrBackpressure
	}
	f.buffer.Add(samples)
	f.registry.Add(metric.MetricForwarded, float64(len(samples)))
	if f.buffer.Len() >= f.batchSize {
		return f.Flush(ctx)
	}
	return nil
}

// Flush drains the buffer and writes all pending samples to the sink.
func (f *Forwarder) Flush(ctx context.Context) error {
	pending := f.buffer.Drain()
	if len(pending) == 0 {
		return nil
	}
	for _, batch := range Assemble(pending, f.batchSize) {
		if err := f.sink.Send(ctx, batch); err != nil {
			f.registry.Add(metric.MetricForwardDropped, float64(len(batch)))
			return err
		}
	}
	return nil
}
