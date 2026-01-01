package stats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

func TestFetchStats(t *testing.T) {
	payload := `{
  "tasks": {"READY": 3, "RUNNING": 2, "FAILED": 1},
  "queues": {
    "default": {"READY": 2, "RUNNING": 1},
    "fast": {"READY": 1, "FAILED": 1}
  },
  "queue_controls": [{
    "queue_name": "fast",
    "paused": true,
    "paused_at": "2024-01-01T12:00:00Z",
    "reason": "maintenance",
    "updated_at": "2024-01-01T12:01:00Z",
    "database": "queues"
  }],
  "worker_health": {"alive": 1, "dead": 0},
  "workers": [{
    "worker_id": "w1",
    "hostname": "host",
    "concurrency": 4,
    "queues": ["default", "fast"],
    "last_seen_at": "2024-01-01T12:00:00Z",
    "version": "0.0.134"
  }],
  "periodic": [{
    "name": "cleanup",
    "cron_expr": "*/5 * * * *",
    "enabled": true,
    "next_run_at": "2024-01-01T12:05:00Z"
  }],
  "databases": [{
    "alias": "default",
    "tasks": {"READY": 3},
    "queues": {"default": {"READY": 3}},
    "workers": [],
    "periodic": []
  }]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stats, err := Fetch(ctx, httpClient, server.URL)
	if err != nil {
		t.Fatalf("fetch stats: %v", err)
	}
	if stats.Tasks["READY"] != 3 {
		t.Fatalf("expected ready count, got %d", stats.Tasks["READY"])
	}
	if len(stats.Workers) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(stats.Workers))
	}
	if stats.Workers[0].WorkerID != "w1" {
		t.Fatalf("worker id mismatch: %s", stats.Workers[0].WorkerID)
	}
	if stats.Workers[0].LastSeenAt.IsZero() {
		t.Fatalf("expected last_seen_at parsed")
	}
	if len(stats.Periodic) != 1 {
		t.Fatalf("expected 1 periodic task, got %d", len(stats.Periodic))
	}
	if stats.Periodic[0].NextRunAt.IsZero() {
		t.Fatalf("expected next_run_at parsed")
	}
	if len(stats.Queues) != 2 {
		t.Fatalf("expected 2 queues, got %d", len(stats.Queues))
	}
	if stats.Queues["default"]["READY"] != 2 {
		t.Fatalf("expected default queue ready count")
	}
	if stats.WorkerHealth == nil || stats.WorkerHealth.Alive != 1 {
		t.Fatalf("expected worker health parsed")
	}
	if len(stats.QueueControls) != 1 || stats.QueueControls[0].QueueName != "fast" {
		t.Fatalf("expected queue controls parsed")
	}
	if len(stats.Databases) != 1 || stats.Databases[0].Alias != "default" {
		t.Fatalf("expected databases parsed")
	}
}

func TestFetchStatsBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := Fetch(ctx, httpClient, server.URL); err == nil {
		t.Fatalf("expected error on non-200 status")
	}
}
