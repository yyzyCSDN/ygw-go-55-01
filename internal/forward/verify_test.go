package forward

import (
	"testing"

	"scrapehub/internal/model"
)

func TestForwardBufferSortedByTime(t *testing.T) {
	buffer := NewBuffer(16)
	buffer.Add([]model.MetricSample{model.NewSample("node_a_cpu", 1, 3000)})
	buffer.Add([]model.MetricSample{
		model.NewSample("node_b_cpu", 2, 1000),
		model.NewSample("node_c_cpu", 3, 2000),
	})
	out := buffer.Drain()
	if len(out) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(out))
	}
	for i := 1; i < len(out); i++ {
		if out[i].Timestamp < out[i-1].Timestamp {
			t.Fatalf("forward buffer is not ordered by time at %d: %+v", i, out)
		}
	}
}
