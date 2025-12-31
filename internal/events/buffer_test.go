package events

import (
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/models"
)

func TestBufferAddAndItems(t *testing.T) {
	buf := NewBuffer(2)
	e1 := models.Event{Message: "first", Timestamp: time.Unix(1, 0)}
	e2 := models.Event{Message: "second", Timestamp: time.Unix(2, 0)}
	e3 := models.Event{Message: "third", Timestamp: time.Unix(3, 0)}

	buf.Add(e1)
	buf.Add(e2)
	items := buf.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Message != "first" || items[1].Message != "second" {
		t.Fatalf("unexpected items order: %+v", items)
	}

	buf.Add(e3)
	items = buf.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items after overflow, got %d", len(items))
	}
	if items[0].Message != "second" || items[1].Message != "third" {
		t.Fatalf("unexpected items after overflow: %+v", items)
	}

	items[0].Message = "mutated"
	fresh := buf.Items()
	if fresh[0].Message != "second" {
		t.Fatalf("expected buffer to return a copy, got %+v", fresh)
	}

	buf.Clear()
	if len(buf.Items()) != 0 {
		t.Fatalf("expected buffer to be cleared")
	}
}
