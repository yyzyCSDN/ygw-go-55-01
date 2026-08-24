package metric

import "sync"

// Registry stores the agent's runtime counters and gauges.
type Registry struct {
	mu    sync.RWMutex
	values map[string]float64
}

// NewRegistry creates an empty metrics registry.
func NewRegistry() *Registry {
	return &Registry{values: make(map[string]float64)}
}

// Inc increments a counter by one.
func (r *Registry) Inc(name string) {
	r.Add(name, 1)
}

// Add adds a delta to a counter.
func (r *Registry) Add(name string, delta float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] += delta
}

// Set stores an absolute value for a gauge.
func (r *Registry) Set(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] = value
}

// Get reads the current value of a metric.
func (r *Registry) Get(name string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.values[name]
}

// Snapshot returns a copy of every tracked value.
func (r *Registry) Snapshot() map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]float64, len(r.values))
	for name, value := range r.values {
		out[name] = value
	}
	return out
}
