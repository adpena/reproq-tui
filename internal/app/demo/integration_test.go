package demo

import (
	"context"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/internal/health"
	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/internal/stats"
	"github.com/adpena/reproq-tui/pkg/client"
)

func TestDemoServerMetricsAndHealth(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatalf("start demo server: %v", err)
	}
	defer server.Close(context.Background())

	httpClient := client.New(client.Options{Timeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := metrics.Scrape(ctx, httpClient, server.MetricsURL, metrics.DefaultCatalog()); err != nil {
		t.Fatalf("scrape metrics: %v", err)
	}
	status, err := health.Fetch(ctx, httpClient, server.HealthURL)
	if err != nil && status.CheckedAt.IsZero() {
		t.Fatalf("fetch health: %v", err)
	}
	if _, err := stats.Fetch(ctx, httpClient, server.StatsURL); err != nil {
		t.Fatalf("fetch stats: %v", err)
	}
}
