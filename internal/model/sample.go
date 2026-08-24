package model

import (
	"fmt"
	"sort"
	"strings"
)

// MetricSample is one parsed time series data point.
type MetricSample struct {
	Name      string
	Value     float64
	Timestamp int64
	Labels    map[string]string
	Source    string
}

// NewSample builds a sample with an empty label set.
func NewSample(name string, value float64, timestamp int64) MetricSample {
	return MetricSample{Name: name, Value: value, Timestamp: timestamp, Labels: map[string]string{}}
}

// Key returns a stable identity for deduplication across scrape attempts.
func (s MetricSample) Key() string {
	keys := make([]string, 0, len(s.Labels))
	for label := range s.Labels {
		keys = append(keys, label)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString(s.Name)
	builder.WriteString("{")
	for _, key := range keys {
		fmt.Fprintf(&builder, "%s=%s,", key, s.Labels[key])
	}
	builder.WriteString(fmt.Sprintf("}@%d", s.Timestamp))
	return builder.String()
}

// WithSource returns a copy of the sample tagged with the given source.
func (s MetricSample) WithSource(source string) MetricSample {
	s.Source = source
	return s
}

// SortSamples orders samples by timestamp and then by name.
func SortSamples(samples []MetricSample) {
	sort.SliceStable(samples, func(i, j int) bool {
		if samples[i].Timestamp != samples[j].Timestamp {
			return samples[i].Timestamp < samples[j].Timestamp
		}
		return samples[i].Name < samples[j].Name
	})
}
