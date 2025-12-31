package metrics

import "testing"

func TestParseSelector(t *testing.T) {
	selector := ParseSelector(`reproq_tasks_processed_total{status="failure",queue="default"}`)
	if selector.Name != "reproq_tasks_processed_total" {
		t.Fatalf("unexpected name: %s", selector.Name)
	}
	if selector.Labels["status"] != "failure" {
		t.Fatalf("expected status label")
	}
	if selector.Labels["queue"] != "default" {
		t.Fatalf("expected queue label")
	}
}
