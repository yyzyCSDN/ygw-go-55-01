package model

import "fmt"

// HealthState is the lifecycle of a target as seen by the health judge.
type HealthState int

const (
	// HealthUnknown is the initial state before any scrape result is known.
	HealthUnknown HealthState = iota
	// HealthHealthy means the last scrape succeeded.
	HealthHealthy
	// HealthUnhealthy means the target keeps failing.
	HealthUnhealthy
	// HealthRemoved means the target was taken out of the scrape list.
	HealthRemoved
)

// String renders a human readable health state.
func (s HealthState) String() string {
	switch s {
	case HealthUnknown:
		return "unknown"
	case HealthHealthy:
		return "healthy"
	case HealthUnhealthy:
		return "unhealthy"
	case HealthRemoved:
		return "removed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}
