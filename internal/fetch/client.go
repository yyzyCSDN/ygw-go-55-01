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
	body, readErr := io.ReadAll(response.Body)
	if readErr != nil && len(body) == 0 {
		// Nothing arrived before the read failed; treat it as a plain
		// transport failure so the fetcher can retry.
		return nil, readErr
	}
	// The read can fail even when the response already arrived — most often
	// because the scrape deadline fired a few dozen milliseconds late. Keep
	// whatever was read so the fetcher can decide whether to honour it as a
	// late success instead of throwing away good data and re-scraping.
	return &Response{
		Body:        body,
		Status:      response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
	}, readErr
}

// CloseResponse releases a response body that was handed back with an error.
func CloseResponse(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}
