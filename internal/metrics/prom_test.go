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

func TestScrapeParsesMetrics(t *testing.T) {
	payload := `# HELP reproq_queue_depth Queue depth
# TYPE reproq_queue_depth gauge
reproq_queue_depth 12
# HELP reproq_tasks_processed_total Total tasks processed
# TYPE reproq_tasks_processed_total counter
reproq_tasks_processed_total{status="success",queue="default"} 95
reproq_tasks_processed_total{status="failure",queue="default"} 5
# HELP reproq_exec_duration_seconds Task execution duration
# TYPE reproq_exec_duration_seconds histogram
reproq_exec_duration_seconds_bucket{le="0.1"} 50
reproq_exec_duration_seconds_bucket{le="0.2"} 80
reproq_exec_duration_seconds_bucket{le="0.5"} 95
reproq_exec_duration_seconds_bucket{le="1"} 100
reproq_exec_duration_seconds_bucket{le="+Inf"} 100
reproq_exec_duration_seconds_sum 12
reproq_exec_duration_seconds_count 100
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
	if got := snapshot.Values[MetricQueueDepth]; got != 12 {
		t.Fatalf("queue depth mismatch: got %v", got)
	}
	if got := snapshot.Values[MetricTasksTotal]; got != 100 {
		t.Fatalf("tasks total mismatch: got %v", got)
	}
	if got := snapshot.Values[MetricTasksFailed]; got != 5 {
		t.Fatalf("tasks failed mismatch: got %v", got)
	}
	if got := snapshot.Values[MetricLatencyP95]; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("p95 mismatch: got %v", got)
	}
}
