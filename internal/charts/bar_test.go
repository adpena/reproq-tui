package charts

import "testing"

func TestBarEmpty(t *testing.T) {
	out := Bar(nil, 4, 2)
	if out == "" {
		t.Fatalf("expected non-empty output")
	}
}
