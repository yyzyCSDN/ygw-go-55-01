package parse

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"time"

	"scrapehub/internal/model"
)

// ErrEmptyLine is returned when a line carries no parseable sample.
var ErrEmptyLine = errors.New("empty metric line")

// Parser converts a scrape body in the agent's text format into samples.
// Every non-comment line must look like: name value [timestamp]
type Parser struct {
	now func() int64
}

// NewParser creates a parser using the system clock for missing timestamps.
func NewParser() *Parser {
	return &Parser{now: func() int64 { return time.Now().UnixMilli() }}
}

// ParseBody parses a complete response body.
func (p *Parser) ParseBody(body []byte, source string) ([]model.MetricSample, error) {
	var samples []model.MetricSample
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sample, err := p.parseLine(line, source, p.now())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		samples = append(samples, sample)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return samples, nil
}

// parseLine turns one text line into a metric sample.
func (p *Parser) parseLine(line, source string, defaultTimestamp int64) (model.MetricSample, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return model.MetricSample{}, ErrEmptyLine
	}
	name, labels, err := splitNameAndLabels(fields[0])
	if err != nil {
		return model.MetricSample{}, err
	}
	value, err := ParseValue(fields[1])
	if err != nil {
		return model.MetricSample{}, fmt.Errorf("invalid value %q: %w", fields[1], err)
	}
	timestamp := defaultTimestamp
	if len(fields) >= 3 {
		timestamp, err = ParseTimestamp(fields[2], defaultTimestamp)
		if err != nil {
			return model.MetricSample{}, fmt.Errorf("invalid timestamp %q: %w", fields[2], err)
		}
	}
	sample := model.NewSample(name, value, timestamp).WithSource(source)
	sample.Labels = labels
	return sample, nil
}
