package fetch

import (
	"context"
	"io"
	"net/http"
	"testing"

	"scrapehub/internal/model"
)

type verifyTrackingBody struct {
	closed bool
}

func (b *verifyTrackingBody) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (b *verifyTrackingBody) Close() error {
	b.closed = true
	return nil
}

type verifyTrackingRT struct {
	body *verifyTrackingBody
}

func (r *verifyTrackingRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: r.body}, nil
}

func TestFetchConnClosedAfterScrape(t *testing.T) {
	body := &verifyTrackingBody{}
	client := NewHTTPClient(&verifyTrackingRT{body: body})
	if _, err := client.Scrape(context.Background(), model.Target{ID: "a", Endpoint: "http://a/metrics"}); err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if !body.closed {
		t.Fatal("scrape response body was not closed, connection leaks")
	}
}
