package metrics

import (
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/models"
)

func TestRingBuffer(t *testing.T) {
	buf := NewRingBuffer(3)
	buf.Add(sampleAt(1, 10))
	buf.Add(sampleAt(2, 20))
	buf.Add(sampleAt(3, 30))
	buf.Add(sampleAt(4, 40))

	values := buf.Values()
	if len(values) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(values))
	}
	if values[0].Value != 20 || values[2].Value != 40 {
		t.Fatalf("unexpected order: %#v", values)
	}
}

func TestRingBufferValuesSince(t *testing.T) {
	buf := NewRingBuffer(5)
	base := time.Unix(100, 0)
	buf.Add(models.Sample{Timestamp: base.Add(-10 * time.Second), Value: 1})
	buf.Add(models.Sample{Timestamp: base.Add(-6 * time.Second), Value: 2})
	buf.Add(models.Sample{Timestamp: base.Add(-2 * time.Second), Value: 3})
	buf.Add(models.Sample{Timestamp: base.Add(-1 * time.Second), Value: 4})

	values := buf.ValuesSince(base.Add(-3 * time.Second))
	if len(values) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(values))
	}
	if values[0].Value != 3 || values[1].Value != 4 {
		t.Fatalf("unexpected filtered values: %#v", values)
	}
}

func sampleAt(sec int64, val float64) models.Sample {
	return models.Sample{
		Timestamp: time.Unix(sec, 0),
		Value:     val,
	}
}
