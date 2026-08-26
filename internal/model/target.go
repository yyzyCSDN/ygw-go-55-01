package model

import "time"

// TargetKind describes where a scrape target came from.
type TargetKind int

const (
	// TargetStatic marks a target registered from the static configuration.
	TargetStatic TargetKind = iota
	// TargetDiscovered marks a target picked up from service discovery.
	TargetDiscovered
)

// Target is a single scrape endpoint that the agent is responsible for.
type Target struct {
	ID       string
	Name     string
	Endpoint string
	Kind     TargetKind
	Labels   map[string]string
	Interval time.Duration
}

// Valid reports whether the target carries enough information to be scraped.
func (t Target) Valid() bool {
	return t.ID != "" && t.Endpoint != ""
}
