package ui

import (
	"strings"
	"testing"

	"github.com/adpena/reproq-tui/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestLowMemoryModeExplainer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	cfg.WorkerHealthURL = "http://worker.local/healthz"
	cfg.EventsURL = "http://worker.local/events"

	model := newTestModel(t, cfg)
	model.lowMemoryMode = true

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = updated.(*Model)

	view := model.View()
	if !strings.Contains(view, "Low memory mode") {
		t.Fatalf("expected low memory explainer, got view: %s", view)
	}
	if !strings.Contains(view, "Events are disabled") {
		t.Fatalf("expected events disabled notice, got view: %s", view)
	}
}
