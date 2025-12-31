package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/internal/config"
	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardViewGolden(t *testing.T) {
	t.Setenv("REPROQ_TUI_SAFE_TOP", "0")
	t.Setenv("REPROQ_TUI_AUTH_TOKEN", "")
	t.Setenv("METRICS_AUTH_TOKEN", "")
	t.Setenv("REPROQ_TUI_AUTH_FILE", filepath.Join(t.TempDir(), "auth.json"))
	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	cfg.WorkerHealthURL = "http://worker.local/healthz"
	cfg.EventsURL = "http://worker.local/events"
	cfg.DjangoStatsURL = "https://django.local/reproq/stats/"

	model := newTestModel(t, cfg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	model = updated.(*Model)

	base := time.Date(2024, 1, 1, 9, 20, 0, 0, time.UTC)
	addSamples := func(key string, values []float64) {
		buf := model.series[key]
		for i, v := range values {
			buf.Add(models.Sample{Timestamp: base.Add(time.Duration(i) * time.Second), Value: v})
		}
	}

	addSamples(metrics.MetricQueueDepth, []float64{3, 5, 4, 6, 7})
	addSamples(metrics.MetricTasksRunning, []float64{2, 3, 2, 2, 3})
	addSamples(metrics.MetricWorkerCount, []float64{2, 3, 3, 3, 3})
	addSamples(metrics.MetricConcurrencyInUse, []float64{2, 3, 2, 2, 3})
	addSamples(metrics.MetricConcurrencyLimit, []float64{10, 10, 10, 10, 10})
	addSamples(metrics.MetricLatencyP95, []float64{0.18, 0.2, 0.22, 0.19, 0.21})
	addSamples(metrics.MetricTasksTotal, []float64{100, 120, 140, 160, 180})
	addSamples(metrics.MetricTasksFailed, []float64{2, 3, 4, 5, 6})
	addSamples(seriesThroughput, []float64{2.5, 3.0, 3.5, 4.0, 4.2})
	addSamples(seriesErrors, []float64{0.05, 0.08, 0.1, 0.12, 0.11})

	model.lastHealth = models.HealthStatus{
		Healthy: true,
		Status:  "ok",
		Version: "0.1.0",
		Build:   "demo",
	}
	model.lastScrapeAt = base.Add(10 * time.Second)
	model.lastScrapeDelay = 120 * time.Millisecond

	model.lastStats = models.DjangoStats{
		Tasks: map[string]int64{
			"READY":   5,
			"WAITING": 3,
			"RUNNING": 2,
			"FAILED":  1,
		},
		Queues: map[string]map[string]int64{
			"default": {
				"READY":   5,
				"RUNNING": 2,
			},
		},
		Workers: []models.WorkerInfo{
			{
				WorkerID:    "w-1",
				Hostname:    "host-a",
				Concurrency: 4,
				Queues:      []string{"default"},
				LastSeenAt:  base,
				Version:     "0.1.0",
			},
		},
		Periodic: []models.PeriodicTask{
			{
				Name:      "nightly",
				CronExpr:  "0 0 * * *",
				Enabled:   true,
				NextRunAt: base.Add(2 * time.Hour),
			},
		},
		FetchedAt: base,
	}

	model.eventsBuffer.Add(models.Event{
		Timestamp: time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
		Level:     "error",
		Message:   "task failed",
	})
	model.eventsBuffer.Add(models.Event{
		Timestamp: time.Date(2024, 1, 1, 9, 31, 0, 0, time.UTC),
		Level:     "warn",
		Message:   "retry scheduled",
	})

	view := normalizeView(model.View())
	goldenPath := filepath.Join("testdata", "dashboard.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(view), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if view != string(data) {
		line, expected, actual := firstDiffLine(string(data), view)
		t.Fatalf("dashboard view does not match golden output (set UPDATE_GOLDEN=1 to refresh)\nline %d\nexpected: %q\nactual:   %q", line, expected, actual)
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var agoRe = regexp.MustCompile(`\b(?:just now|\d+[smh] ago)\b`)

func normalizeView(input string) string {
	withoutAnsi := ansiRe.ReplaceAllString(input, "")
	replacer := strings.NewReplacer(
		"╭", "+", "╮", "+", "╰", "+", "╯", "+",
		"┌", "+", "┐", "+", "└", "+", "┘", "+",
		"─", "-", "│", "|", "├", "+", "┤", "+",
		"┬", "+", "┴", "+", "┼", "+",
		"█", "#", "░", ".", "▁", "*", "▂", "*", "▃", "*", "▄", "*",
		"▅", "*", "▆", "*", "▇", "*", "•", "*",
	)
	withoutAnsi = replacer.Replace(withoutAnsi)
	withoutAnsi = agoRe.ReplaceAllString(withoutAnsi, "<ago>")
	lines := strings.Split(withoutAnsi, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func firstDiffLine(expected, actual string) (int, string, string) {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")
	max := len(expLines)
	if len(actLines) > max {
		max = len(actLines)
	}
	for i := 0; i < max; i++ {
		expLine := ""
		actLine := ""
		if i < len(expLines) {
			expLine = expLines[i]
		}
		if i < len(actLines) {
			actLine = actLines[i]
		}
		if expLine != actLine {
			return i + 1, expLine, actLine
		}
	}
	return max, "", ""
}
