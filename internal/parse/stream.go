package parse

import (
	"bytes"

	"scrapehub/internal/model"
)

// StreamReader feeds response chunks into the parser and keeps the partial
// tail of a line across chunks so a resumed stream never starts mid-line.
type StreamReader struct {
	parser *Parser
	source string
	carry  []byte
}

// NewStreamReader creates a reader for one scrape response.
func NewStreamReader(parser *Parser, source string) *StreamReader {
	return &StreamReader{parser: parser, source: source}
}

// Feed consumes a chunk. When final is true the remaining carry is flushed
// as a complete line.
func (r *StreamReader) Feed(chunk []byte, final bool) ([]model.MetricSample, error) {
	lines := bytes.Split(chunk, []byte{'\n'})
	complete := lines
	if !final {
		complete = lines[:len(lines)-1]
	}
	var samples []model.MetricSample
	for _, line := range complete {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		sample, err := r.parser.parseLine(string(trimmed), r.source, r.parser.now())
		if err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return samples, nil
}
