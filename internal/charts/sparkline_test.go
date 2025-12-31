package charts

import (
	"math"
	"testing"
)

func TestSparklineEmpty(t *testing.T) {
	out := Sparkline(nil, 5)
	if len([]rune(out)) != 5 {
		t.Fatalf("expected width 5, got %q", out)
	}
}

func TestSparklineNaN(t *testing.T) {
	values := []float64{math.NaN(), math.NaN(), math.NaN()}
	out := Sparkline(values, 3)
	if len([]rune(out)) != 3 {
		t.Fatalf("expected width 3, got %q", out)
	}
}

func TestSparklineSpikes(t *testing.T) {
	values := []float64{1, 100, 2, 80, 3, 60}
	out := Sparkline(values, 6)
	if len([]rune(out)) != 6 {
		t.Fatalf("expected width 6, got %q", out)
	}
}

func TestSparklineSinglePoint(t *testing.T) {
	values := []float64{3.14}
	out := Sparkline(values, 4)
	if len([]rune(out)) != 4 {
		t.Fatalf("expected width 4, got %q", out)
	}
}

func TestSparklineInf(t *testing.T) {
	values := []float64{math.Inf(1), 1}
	out := Sparkline(values, 2)
	if len([]rune(out)) != 2 {
		t.Fatalf("expected width 2, got %q", out)
	}
}
