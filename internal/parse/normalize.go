package parse

import (
	"fmt"
	"strconv"
	"strings"
)

// NormalizeName validates and canonicalizes a metric name.
func NormalizeName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	for _, r := range name {
		if !(r == '_' || r == ':' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return ""
		}
	}
	return name
}

// splitNameAndLabels parses a token like `cpu_usage{host="a",dc="cn"}` into a
// metric name and its label set.
func splitNameAndLabels(raw string) (string, map[string]string, error) {
	raw = strings.TrimSpace(raw)
	open := strings.IndexByte(raw, '{')
	if open < 0 {
		name := NormalizeName(raw)
		if name == "" {
			return "", nil, fmt.Errorf("invalid metric name %q", raw)
		}
		return name, map[string]string{}, nil
	}
	if !strings.HasSuffix(raw, "}") {
		return "", nil, fmt.Errorf("unterminated label block in %q", raw)
	}
	name := NormalizeName(raw[:open])
	if name == "" {
		return "", nil, fmt.Errorf("invalid metric name %q", raw[:open])
	}
	labels, err := parseLabels(raw[open+1 : len(raw)-1])
	if err != nil {
		return "", nil, err
	}
	return name, labels, nil
}

// parseLabels turns `host="a",dc="cn"` into a label map.
func parseLabels(raw string) (map[string]string, error) {
	labels := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return labels, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("label entry %q has no value", part)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		} else {
			return nil, fmt.Errorf("label value %q must be quoted", value)
		}
		if NormalizeName(key) == "" {
			return nil, fmt.Errorf("invalid label name %q", key)
		}
		labels[key] = value
	}
	return labels, nil
}

// ParseValue parses a float metric value.
func ParseValue(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

// ParseTimestamp parses an optional unix millisecond timestamp and falls
// back to the default when the field is absent.
func ParseTimestamp(raw string, fallback int64) (int64, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bad timestamp %q", raw)
	}
	return value, nil
}
