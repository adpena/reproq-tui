package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

func TestFetchConfig(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reproq/tui/config/":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"worker_url":         "https://worker.example.com",
				"worker_metrics_url": "https://worker.example.com/metrics",
				"worker_health_url":  "https://worker.example.com/healthz",
				"events_url":         "https://worker.example.com/events",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	ctx := context.Background()

	cfg, err := FetchConfig(ctx, httpClient, server.URL)
	if err != nil {
		t.Fatalf("fetch config: %v", err)
	}
	if cfg.WorkerURL != "https://worker.example.com" {
		t.Fatalf("expected worker url, got %q", cfg.WorkerURL)
	}
	if cfg.WorkerMetricsURL != "https://worker.example.com/metrics" {
		t.Fatalf("expected metrics url, got %q", cfg.WorkerMetricsURL)
	}
	if cfg.WorkerHealthURL != "https://worker.example.com/healthz" {
		t.Fatalf("expected health url, got %q", cfg.WorkerHealthURL)
	}
	if cfg.EventsURL != "https://worker.example.com/events" {
		t.Fatalf("expected events url, got %q", cfg.EventsURL)
	}
}
