package retry

// AttemptState records what happened during one scrape attempt.
type AttemptState struct {
	Attempt   int
	LastErr   error
	Retryable bool
}

// NewAttemptState initialises a state for the first attempt.
func NewAttemptState() *AttemptState {
	return &AttemptState{}
}

// Record stores the outcome of the latest attempt.
func (s *AttemptState) Record(err error, retryable bool) {
	s.Attempt++
	s.LastErr = err
	s.Retryable = retryable
}

// Done reports whether the attempt sequence should stop.
func (s *AttemptState) Done(maxAttempts int) bool {
	return s.Attempt >= maxAttempts || !s.Retryable
}
