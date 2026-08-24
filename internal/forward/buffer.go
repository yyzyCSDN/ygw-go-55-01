package forward

import "scrapehub/internal/model"

// Buffer holds samples that have not been written downstream yet. Draining
// returns them in timestamp order so the sink always sees a consistent
// timeline.
type Buffer struct {
	capacity int
	samples  []model.MetricSample
}

// NewBuffer creates an empty ordered buffer.
func NewBuffer(capacity int) *Buffer {
	if capacity < 1 {
		capacity = 1024
	}
	return &Buffer{capacity: capacity, samples: make([]model.MetricSample, 0, capacity)}
}

// Add appends samples to the buffer.
func (b *Buffer) Add(samples []model.MetricSample) {
	b.samples = append(b.samples, samples...)
}

// Len returns the number of pending samples.
func (b *Buffer) Len() int {
	return len(b.samples)
}

// Full reports whether the buffer reached its capacity.
func (b *Buffer) Full() bool {
	return len(b.samples) >= b.capacity
}

// Drain returns every pending sample in timestamp order and clears the buffer.
// Multiple targets are scraped concurrently, so samples arrive interleaved by
// completion time rather than by their actual timestamps. Sorting here keeps the
// timeline the downstream sink receives consistent regardless of which scrape
// finished first.
func (b *Buffer) Drain() []model.MetricSample {
	out := b.samples
	b.samples = make([]model.MetricSample, 0, b.capacity)
	model.SortSamples(out)
	return out
}
