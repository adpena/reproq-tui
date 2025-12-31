package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

func TestFetchHealthJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"1.2.3","build":"demo","commit":"abc123","message":"ready"}`))
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	status, err := Fetch(context.Background(), httpClient, server.URL)
	if err != nil {
		t.Fatalf("fetch health: %v", err)
	}
	if !status.Healthy {
		t.Fatalf("expected healthy status, got %+v", status)
	}
	if status.Status != "ok" {
		t.Fatalf("expected status ok, got %q", status.Status)
	}
	if status.Version != "1.2.3" || status.Build != "demo" || status.Commit != "abc123" {
		t.Fatalf("unexpected version info: %+v", status)
	}
	if status.Message != "ready" {
		t.Fatalf("unexpected message: %q", status.Message)
	}
	if status.CheckedAt.IsZero() {
		t.Fatalf("expected checked_at timestamp to be set")
	}
	if status.Latency <= 0 {
		t.Fatalf("expected latency to be set")
	}
}

func TestFetchHealthJSONWithoutContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	status, err := Fetch(context.Background(), httpClient, server.URL)
	if err != nil {
		t.Fatalf("fetch health: %v", err)
	}
	if !status.Healthy {
		t.Fatalf("expected healthy status, got %+v", status)
	}
	if status.Status != "healthy" {
		t.Fatalf("expected status healthy, got %q", status.Status)
	}
}

func TestFetchHealthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	status, err := Fetch(context.Background(), httpClient, server.URL)
	if err == nil {
		t.Fatalf("expected error for unhealthy response")
	}
	if status.Healthy {
		t.Fatalf("expected unhealthy status, got %+v", status)
	}
	if !strings.Contains(status.Status, "service") {
		t.Fatalf("expected status text to reflect http status, got %q", status.Status)
	}
}
