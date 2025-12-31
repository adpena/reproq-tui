package metrics

import (
	"math"
	"time"

	"github.com/adpena/reproq-tui/pkg/models"
)

func Rate(samples []models.Sample) float64 {
	if len(samples) < 2 {
		return math.NaN()
	}
	first := samples[0]
	last := samples[len(samples)-1]
	delta := last.Value - first.Value
	if delta < 0 {
		delta = 0
	}
	elapsed := last.Timestamp.Sub(first.Timestamp).Seconds()
	if elapsed <= 0 {
		return math.NaN()
	}
	return delta / elapsed
}

func Delta(samples []models.Sample) float64 {
	if len(samples) < 2 {
		return math.NaN()
	}
	first := samples[0]
	last := samples[len(samples)-1]
	delta := last.Value - first.Value
	if delta < 0 {
		return 0
	}
	return delta
}

func Ratio(numerator, denominator []models.Sample) float64 {
	n := Delta(numerator)
	d := Delta(denominator)
	if math.IsNaN(n) || math.IsNaN(d) || d == 0 {
		return math.NaN()
	}
	return n / d
}

func WindowCutoff(window time.Duration, now time.Time) time.Time {
	if window <= 0 {
		return time.Time{}
	}
	return now.Add(-window)
}
