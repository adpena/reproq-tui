package charts

import "testing"

func TestGauge(t *testing.T) {
	out := Gauge(5, 10, 10)
	if len([]rune(out)) != 10 {
		t.Fatalf("expected width 10, got %q", out)
	}
}
