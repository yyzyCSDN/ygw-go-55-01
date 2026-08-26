package retry

import "time"

// ComputeBackoff returns an exponential backoff bounded by cap.
func ComputeBackoff(base time.Duration, attempt int, cap time.Duration) time.Duration {
	if base <= 0 {
		base = time.Millisecond
	}
	if attempt < 0 {
		attempt = 0
	}
	delay := base
	for i := 0; i < attempt && delay < cap; i++ {
		delay *= 2
	}
	if delay > cap {
		return cap
	}
	return delay
}
