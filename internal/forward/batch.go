package forward

import "scrapehub/internal/model"

// Assemble splits pending samples into batches no larger than max.
func Assemble(pending []model.MetricSample, max int) [][]model.MetricSample {
	if len(pending) == 0 {
		return nil
	}
	if max < 1 {
		max = 1
	}
	batches := make([][]model.MetricSample, 0, (len(pending)+max-1)/max)
	for start := 0; start < len(pending); start += max {
		end := start + max
		if end > len(pending) {
			end = len(pending)
		}
		batches = append(batches, pending[start:end])
	}
	return batches
}
