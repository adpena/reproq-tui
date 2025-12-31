package ui

import (
	"context"
	"testing"

	"github.com/adpena/reproq-tui/internal/app/demo"
	"github.com/adpena/reproq-tui/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHeadlessUpdateLoop(t *testing.T) {
	server, err := demo.Start()
	if err != nil {
		t.Fatalf("start demo server: %v", err)
	}
	defer server.Close(context.Background())

	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = server.MetricsURL
	cfg.WorkerHealthURL = server.HealthURL
	cfg.EventsURL = ""
	cfg.DjangoStatsURL = server.StatsURL

	model := newTestModel(t, cfg)
	defer model.Close()

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = updated.(*Model)

	msg := pollMetricsCmd(cfg, model.client, model.catalog)()
	if msg != nil {
		updated, _ := model.Update(msg)
		model = updated.(*Model)
	}
	msg = pollHealthCmd(cfg, model.client)()
	if msg != nil {
		updated, _ := model.Update(msg)
		model = updated.(*Model)
	}
	msg = pollStatsCmd(cfg, model.client)()
	if msg != nil {
		updated, _ := model.Update(msg)
		model = updated.(*Model)
	}

	view := model.View()
	if view == "" {
		t.Fatalf("expected non-empty view")
	}

	updated, _ = model.Update(metricsTickMsg{})
	model = updated.(*Model)
	updated, _ = model.Update(healthTickMsg{})
	model = updated.(*Model)
	_, _ = model.Update(statsTickMsg{})
}
