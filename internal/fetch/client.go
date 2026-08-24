package fetch

import (
	"context"
	"io"
	"net/http"

	"scrapehub/internal/model"
)

// RoundTripper is the transport seam used by the HTTP client.
type RoundTripper interface {
	RoundTrip(req *http.Request) (*http.Response, error)
}

// Response is the readable outcome of one scrape request.
type Response struct {
	Body        []byte
	Status      int
	ContentType string
}

// HTTPClient performs the actual HTTP GET against a target endpoint.
type HTTPClient struct {
	transport RoundTripper
}

// NewHTTPClient builds a client over the given transport.
func NewHTTPClient(transport RoundTripper) *HTTPClient {
	return &HTTPClient{transport: transport}
}

// Scrape executes one GET and reads the full body.
func (c *HTTPClient) Scrape(ctx context.Context, target model.Target) (*Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.transport.RoundTrip(request)
	if err != nil {
		CloseResponse(response)
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		Body:        body,
		Status:      response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
	}, nil
}

// CloseResponse releases a response body that was handed back with an error.
func CloseResponse(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}
