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
	r.carry = append(r.carry, chunk...)
	lines := bytes.Split(r.carry, []byte{'\n'})
	// The last piece is only complete when the stream is final.
	complete := lines[:len(lines)-1]
	r.carry = lines[len(lines)-1]
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
	if final && len(bytes.TrimSpace(r.carry)) > 0 {
		line := bytes.TrimSpace(r.carry)
		if line[0] != '#' {
			sample, err := r.parser.parseLine(string(line), r.source, r.parser.now())
			if err != nil {
				return nil, err
			}
			samples = append(samples, sample)
		}
		r.carry = nil
	}
	return samples, nil
}
