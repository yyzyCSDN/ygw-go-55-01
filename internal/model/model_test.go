package model

import "testing"

func TestTaskTransitionsValid(t *testing.T) {
	task := NewTask("target-a", 7)
	if task.State != TaskIdle {
		t.Fatalf("expected idle task, got %s", task.State)
	}
	for _, next := range []TaskState{TaskScheduled, TaskFetching, TaskParsed, TaskForwarded, TaskIdle} {
		if err := task.Transition(next); err != nil {
			t.Fatalf("transition to %s failed: %v", next, err)
		}
	}
}

func TestTaskTransitionInvalid(t *testing.T) {
	task := NewTask("target-b", 1)
	if err := task.Transition(TaskParsed); err == nil {
		t.Fatal("expected invalid transition from idle to parsed to fail")
	}
}

func TestSampleKeyStable(t *testing.T) {
	first := NewSample("cpu_usage", 1.5, 1000).WithSource("node-a")
	second := NewSample("cpu_usage", 1.5, 1000).WithSource("node-a")
	if first.Key() != second.Key() {
		t.Fatalf("stable samples produced different keys: %q vs %q", first.Key(), second.Key())
	}
	third := NewSample("cpu_usage", 1.5, 1001).WithSource("node-a")
	if first.Key() == third.Key() {
		t.Fatal("samples with different timestamps must have different keys")
	}
}

func TestSortSamplesByTimestamp(t *testing.T) {
	samples := []MetricSample{
		NewSample("b", 1, 3000),
		NewSample("a", 1, 1000),
		NewSample("c", 1, 2000),
	}
	SortSamples(samples)
	for i := 1; i < len(samples); i++ {
		if samples[i].Timestamp < samples[i-1].Timestamp {
			t.Fatalf("samples not sorted by timestamp at %d", i)
		}
	}
}
