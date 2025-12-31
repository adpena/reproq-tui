package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adpena/reproq-tui/internal/config"
)

func newTestModel(t *testing.T, cfg config.Config) *Model {
	t.Helper()
	if os.Getenv("REPROQ_TUI_SAFE_TOP") == "" {
		t.Setenv("REPROQ_TUI_SAFE_TOP", "0")
	}
	if os.Getenv("REPROQ_TUI_AUTH_TOKEN") == "" {
		t.Setenv("REPROQ_TUI_AUTH_TOKEN", "")
	}
	if os.Getenv("METRICS_AUTH_TOKEN") == "" {
		t.Setenv("METRICS_AUTH_TOKEN", "")
	}
	if os.Getenv("REPROQ_TUI_AUTH_FILE") == "" {
		t.Setenv("REPROQ_TUI_AUTH_FILE", filepath.Join(t.TempDir(), "auth.json"))
	}
	return NewModel(cfg)
}
