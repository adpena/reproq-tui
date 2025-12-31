package events

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
)

func TestParseEventPayload(t *testing.T) {
	ts := "2024-01-01T12:00:00Z"
	payload := `{"ts":"` + ts + `","level":"warn","type":"task_failed","msg":"boom","queue":"default","task_id":"42","worker_id":"w1","metadata":{"attempt":"1","latency":"250"}}`
	event, ok := parseEvent(payload)
	if !ok {
		t.Fatalf("expected payload to parse")
	}
	if event.Level != "warn" || event.Type != "task_failed" || event.Message != "boom" {
		t.Fatalf("unexpected event fields: %+v", event)
	}
	if event.Queue != "default" || event.TaskID != "42" || event.WorkerID != "w1" {
		t.Fatalf("unexpected event ids: %+v", event)
	}
	if event.Metadata["attempt"] != "1" || event.Metadata["latency"] != "250" {
		t.Fatalf("unexpected metadata: %+v", event.Metadata)
	}
	wantTime, _ := time.Parse(time.RFC3339, ts)
	if !event.Timestamp.Equal(wantTime) {
		t.Fatalf("unexpected timestamp: %v", event.Timestamp)
	}
}

func TestParseEventUnixTimestamp(t *testing.T) {
	payload := `{"ts":1700000000.5,"level":"info","type":"task","msg":"ok"}`
	event, ok := parseEvent(payload)
	if !ok {
		t.Fatalf("expected payload to parse")
	}
	if event.Timestamp.Unix() != 1700000000 {
		t.Fatalf("unexpected unix seconds: %d", event.Timestamp.Unix())
	}
	if event.Timestamp.Nanosecond() == 0 {
		t.Fatalf("expected fractional seconds to be preserved")
	}
}

func TestParseEventInvalid(t *testing.T) {
	if _, ok := parseEvent("{invalid"); ok {
		t.Fatalf("expected invalid payload to fail")
	}
}

func TestConnectStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	err := connect(context.Background(), httpClient, server.URL, make(chan models.Event, 1))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !client.IsStatus(err, http.StatusInternalServerError) {
		t.Fatalf("expected status error, got %v", err)
	}
}
