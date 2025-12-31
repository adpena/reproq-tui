package metrics

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

func TestCatalogOverrides(t *testing.T) {
	catalog := NewCatalog(map[string]string{
		MetricQueueDepth: "custom_queue_depth",
	})
	if got := catalog.Name(MetricQueueDepth); got != "custom_queue_depth" {
		t.Fatalf("expected override, got %s", got)
	}
	if got := catalog.Name(MetricTasksTotal); got == "" {
		t.Fatalf("expected default mapping for tasks total")
	}
	selector := catalog.Selectors[MetricTasksFailed]
	if selector.Name != "reproq_tasks_processed_total" {
		t.Fatalf("expected selector name, got %s", selector.Name)
	}
	if selector.Labels["status"] != "failure" {
		t.Fatalf("expected failure label selector")
	}
}

func TestScrapeMissingMetricReturnsNaN(t *testing.T) {
	payload := `# HELP reproq_queue_depth Queue depth
# TYPE reproq_queue_depth gauge
reproq_queue_depth 12
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	snapshot, err := Scrape(ctx, httpClient, server.URL, DefaultCatalog())
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if !math.IsNaN(snapshot.Values[MetricTasksTotal]) {
		t.Fatalf("expected NaN for missing metric")
	}
}
