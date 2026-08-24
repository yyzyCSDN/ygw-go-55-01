package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"scrapehub/internal/fetch"
	"scrapehub/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// wrapClient adapts the concrete HTTP transport client to the fetch Client
// interface used by the fetcher.
func wrapClient(client *fetch.HTTPClient) fetch.Client {
	return clientAdapter{inner: client}
}

type clientAdapter struct {
	inner *fetch.HTTPClient
}

func (a clientAdapter) Scrape(ctx context.Context, target model.Target) fetch.Outcome {
	response, err := a.inner.Scrape(ctx, target)
	return fetch.Outcome{Response: response, Err: err}
}
