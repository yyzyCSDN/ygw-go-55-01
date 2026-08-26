package discover

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"scrapehub/internal/model"
)

// sdPayload is the JSON shape returned by the discovery endpoint.
type sdPayload struct {
	Targets []struct {
		ID       string            `json:"id"`
		Name     string            `json:"name"`
		Endpoint string            `json:"endpoint"`
		Labels   map[string]string `json:"labels"`
	} `json:"targets"`
}

// HTTPSDProvider pulls the target list from a service discovery HTTP endpoint.
type HTTPSDProvider struct {
	endpoint string
	client   *http.Client
	mu       sync.RWMutex
	targets  map[string]model.Target
}

// NewHTTPSDProvider builds a provider for the given discovery endpoint.
func NewHTTPSDProvider(endpoint string, client *http.Client) *HTTPSDProvider {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPSDProvider{endpoint: endpoint, client: client, targets: make(map[string]model.Target)}
}

// Refresh fetches the latest target list from the discovery endpoint.
func (p *HTTPSDProvider) Refresh() ([]model.Target, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.client.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var payload sdPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	next := make(map[string]model.Target, len(payload.Targets))
	for _, item := range payload.Targets {
		if item.ID == "" || item.Endpoint == "" {
			continue
		}
		next[item.ID] = model.Target{
			ID:       item.ID,
			Name:     item.Name,
			Endpoint: item.Endpoint,
			Kind:     model.TargetDiscovered,
			Labels:   item.Labels,
		}
	}
	p.mu.Lock()
	p.targets = next
	p.mu.Unlock()
	return p.List(), nil
}

// List returns the targets fetched by the last refresh.
func (p *HTTPSDProvider) List() []model.Target {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]model.Target, 0, len(p.targets))
	for _, target := range p.targets {
		out = append(out, target)
	}
	return sortTargets(out)
}
