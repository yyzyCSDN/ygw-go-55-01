package forward

import (
	"context"
	"testing"

	"scrapehub/internal/metric"
	"scrapehub/internal/model"
)

func TestBufferAddDrainCount(t *testing.T) {
	buffer := NewBuffer(8)
	buffer.Add([]model.MetricSample{model.NewSample("a", 1, 1000)})
	buffer.Add([]model.MetricSample{
		model.NewSample("b", 2, 1000),
		model.NewSample("c", 3, 1000),
	})
	if buffer.Len() != 3 {
		t.Fatalf("expected 3 pending samples, got %d", buffer.Len())
	}
	drained := buffer.Drain()
	if len(drained) != 3 {
		t.Fatalf("expected 3 drained samples, got %d", len(drained))
	}
	if buffer.Len() != 0 {
		t.Fatal("buffer should be empty after drain")
	}
}

func TestAssembleBatches(t *testing.T) {
	samples := make([]model.MetricSample, 5)
	for i := range samples {
		samples[i] = model.NewSample("s", float64(i), int64(i))
	}
	batches := Assemble(samples, 2)
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 || len(batches[2]) != 1 {
		t.Fatalf("unexpected batch sizes: %d,%d,%d", len(batches[0]), len(batches[1]), len(batches[2]))
	}
}

func TestForwarderFlushesAtBatchSize(t *testing.T) {
	registry := metric.NewRegistry()
	sink := &recordingSink{}
	forwarder := NewForwarder(NewBuffer(8), sink, 2, registry)
	ctx := context.Background()
	if err := forwarder.Forward(ctx, []model.MetricSample{model.NewSample("a", 1, 1000)}); err != nil {
		t.Fatalf("forward failed: %v", err)
	}
	if sink.batches != 0 {
		t.Fatal("batch should not flush before threshold")
	}
	if err := forwarder.Forward(ctx, []model.MetricSample{model.NewSample("b", 2, 1000)}); err != nil {
		t.Fatalf("forward failed: %v", err)
	}
	if sink.batches != 1 || sink.samples != 2 {
		t.Fatalf("expected one flushed batch of 2, got batches=%d samples=%d", sink.batches, sink.samples)
	}
}

type recordingSink struct {
	batches int
	samples int
}

func (s *recordingSink) Send(_ context.Context, samples []model.MetricSample) error {
	s.batches++
	s.samples += len(samples)
	return nil
}
