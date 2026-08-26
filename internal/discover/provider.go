package discover

import (
	"sync"

	"scrapehub/internal/model"
)

// Provider yields the current set of scrape targets.
type Provider interface {
	// List returns the targets known right now.
	List() []model.Target
	// Refresh asks the provider to reload its target list.
	Refresh() ([]model.Target, error)
}

// StaticProvider keeps an in-memory target list that can be edited at runtime.
type StaticProvider struct {
	mu      sync.RWMutex
	targets map[string]model.Target
}

// NewStaticProvider builds a provider from the initial target set.
func NewStaticProvider(targets ...model.Target) *StaticProvider {
	provider := &StaticProvider{targets: make(map[string]model.Target, len(targets))}
	for _, target := range targets {
		if target.Valid() {
			provider.targets[target.ID] = target
		}
	}
	return provider
}

// List returns a copy of the current targets sorted by ID.
func (p *StaticProvider) List() []model.Target {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]model.Target, 0, len(p.targets))
	for _, target := range p.targets {
		out = append(out, target)
	}
	return sortTargets(out)
}

// Add registers or replaces a target.
func (p *StaticProvider) Add(target model.Target) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target.ID] = target
}

// Remove deletes a target from the provider.
func (p *StaticProvider) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.targets, id)
}

// Refresh for a static provider simply re-reads the in-memory list.
func (p *StaticProvider) Refresh() ([]model.Target, error) {
	return p.List(), nil
}
