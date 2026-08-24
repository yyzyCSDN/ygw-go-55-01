package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"scrapehub/internal/model"
)

// Sink accepts batches of samples from the forwarder.
type Sink interface {
	Send(ctx context.Context, samples []model.MetricSample) error
}

// HTTPSink posts sample batches to a downstream HTTP endpoint.
type HTTPSink struct {
	endpoint string
	client   *http.Client
}

// NewHTTPSink builds a sink for the given endpoint.
func NewHTTPSink(endpoint string, timeout time.Duration) *HTTPSink {
	return &HTTPSink{
		endpoint: endpoint,
		client:   &http.Client{Timeout: timeout},
	}
}

// Send serialises the batch and posts it to the downstream endpoint.
func (s *HTTPSink) Send(ctx context.Context, samples []model.MetricSample) error {
	payload, err := json.Marshal(map[string]any{
		"samples": samples,
		"count":   len(samples),
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("downstream returned %s", response.Status)
	}
	return nil
}
