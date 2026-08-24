package retry

import (
	"errors"
	"testing"
	"time"
)

func TestBackoffGrows(t *testing.T) {
	first := ComputeBackoff(time.Millisecond, 0, time.Second)
	second := ComputeBackoff(time.Millisecond, 1, time.Second)
	if second <= first {
		t.Fatalf("backoff should grow: %v then %v", first, second)
	}
	capped := ComputeBackoff(time.Millisecond, 100, time.Second)
	if capped != time.Second {
		t.Fatalf("backoff should be capped at 1s, got %v", capped)
	}
}

func TestPolicyPermanentNotRetried(t *testing.T) {
	policy := NewPolicy(3, time.Millisecond, time.Millisecond, nil)
	if policy.ShouldRetry(ErrPermanent, 0) {
		t.Fatal("permanent errors must not be retried")
	}
	if policy.ShouldRetry(nil, 0) {
		t.Fatal("nil errors must not be retried")
	}
	if policy.ShouldRetry(ErrTimeout, 3) {
		t.Fatal("attempt past max must not be retried")
	}
	if !policy.ShouldRetry(ErrTimeout, 1) {
		t.Fatal("transient timeout should be retried")
	}
}

func TestAttemptStateDone(t *testing.T) {
	state := NewAttemptState()
	state.Record(errors.New("boom"), true)
	if state.Done(3) {
		t.Fatal("state should not be done after first attempt")
	}
	state.Record(errors.New("boom"), true)
	state.Record(errors.New("boom"), true)
	if !state.Done(3) {
		t.Fatal("state should be done after max attempts")
	}
}
