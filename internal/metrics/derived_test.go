package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/models"
)

func TestRate(t *testing.T) {
	samples := []models.Sample{
		{Timestamp: time.Unix(0, 0), Value: 10},
		{Timestamp: time.Unix(2, 0), Value: 14},
	}
	rate := Rate(samples)
	if math.Abs(rate-2.0) > 0.001 {
		t.Fatalf("expected rate 2.0, got %v", rate)
	}
}

func TestRatio(t *testing.T) {
	numerator := []models.Sample{
		{Timestamp: time.Unix(0, 0), Value: 2},
		{Timestamp: time.Unix(2, 0), Value: 4},
	}
	denominator := []models.Sample{
		{Timestamp: time.Unix(0, 0), Value: 10},
		{Timestamp: time.Unix(2, 0), Value: 20},
	}
	ratio := Ratio(numerator, denominator)
	if math.Abs(ratio-0.2) > 0.0001 {
		t.Fatalf("expected ratio 0.2, got %v", ratio)
	}
}
