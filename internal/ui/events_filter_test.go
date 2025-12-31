package ui

import "testing"

func TestParseSSEFilter(t *testing.T) {
	filter := parseSSEFilter("queue:default error worker=w1 task=42")
	if filter.queue != "default" {
		t.Fatalf("expected queue filter, got %q", filter.queue)
	}
	if filter.workerID != "w1" {
		t.Fatalf("expected worker filter, got %q", filter.workerID)
	}
	if filter.taskID != "42" {
		t.Fatalf("expected task filter, got %q", filter.taskID)
	}
	if filter.local != "error" {
		t.Fatalf("expected local filter 'error', got %q", filter.local)
	}
}

func TestBuildEventsURL(t *testing.T) {
	filter := sseFilter{queue: "default", workerID: "w1", taskID: "42"}
	updated := buildEventsURL("https://example.com/events?foo=bar", filter)
	if updated == "" {
		t.Fatalf("expected url")
	}
	if updated == "https://example.com/events?foo=bar" {
		t.Fatalf("expected query params updated, got %s", updated)
	}
	reset := buildEventsURL(updated, sseFilter{})
	if reset == "" || reset == updated {
		t.Fatalf("expected reset query, got %s", reset)
	}
}
