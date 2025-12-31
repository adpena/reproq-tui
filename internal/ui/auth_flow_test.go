package ui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/internal/auth"
	"github.com/adpena/reproq-tui/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAuthURLPromptLifecycle(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	t.Setenv("REPROQ_TUI_AUTH_FILE", authPath)

	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	model := NewModel(cfg)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	model = updated.(*Model)

	if !model.authURLActive {
		t.Fatalf("expected auth URL prompt to be active")
	}

	model.authURLInput.SetValue("http://")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)

	if !model.authURLActive {
		t.Fatalf("expected prompt to remain active on invalid url")
	}
	if model.authURLNotice == "" {
		t.Fatalf("expected validation notice on invalid url")
	}
	if model.authFlowActive {
		t.Fatalf("did not expect auth flow to start on invalid url")
	}

	model.authURLInput.SetValue("django.example.com\\reproq\\stats\\")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)

	if model.authURLActive {
		t.Fatalf("expected prompt to close on valid url")
	}
	if model.cfg.DjangoURL != "https://django.example.com" {
		t.Fatalf("unexpected django url: %s", model.cfg.DjangoURL)
	}
	if model.cfg.DjangoStatsURL != "https://django.example.com/reproq/stats/" {
		t.Fatalf("unexpected stats url: %s", model.cfg.DjangoStatsURL)
	}

	updated, _ = model.Update(tuiConfigMsg{cfg: auth.TUIConfig{}})
	model = updated.(*Model)
	if !model.authFlowActive {
		t.Fatalf("expected auth flow to start after config fetch")
	}
}

func TestAuthCancelDuringFlow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	model := NewModel(cfg)
	model.authFlowActive = true
	model.authPair = auth.Pairing{Code: "abc"}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(*Model)

	if model.authFlowActive {
		t.Fatalf("expected auth flow to be canceled")
	}
	if model.authPair.Code != "" {
		t.Fatalf("expected auth pair to be cleared")
	}
	if model.toast != "Auth canceled" {
		t.Fatalf("expected cancel toast, got %q", model.toast)
	}
}

func TestAuthKeyCancelsFlow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	model := NewModel(cfg)
	model.authFlowActive = true
	model.authPair = auth.Pairing{Code: "abc"}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	model = updated.(*Model)

	if model.authFlowActive {
		t.Fatalf("expected auth flow to be canceled")
	}
	if model.authPair.Code != "" {
		t.Fatalf("expected auth pair to be cleared")
	}
	if model.toast != "Auth canceled" {
		t.Fatalf("expected cancel toast, got %q", model.toast)
	}
}

func TestAuthKeyOpensURLPromptWhenMissingDjango(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	model := NewModel(cfg)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	model = updated.(*Model)

	if !model.authURLActive {
		t.Fatalf("expected auth URL prompt to open")
	}
	if model.authURLInput.Value() != "" {
		t.Fatalf("expected empty auth URL input, got %q", model.authURLInput.Value())
	}
}

func TestAuthApprovedUpdatesTokenAndPersists(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	t.Setenv("REPROQ_TUI_AUTH_FILE", authPath)

	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	cfg.DjangoURL = "https://django.example.com"
	cfg.DjangoStatsURL = "https://django.example.com/reproq/stats/"
	model := NewModel(cfg)
	model.authFlowActive = true

	expires := time.Now().Add(time.Hour)
	updated, _ := model.Update(authStatusMsg{
		status: auth.PairStatus{
			Status:    "approved",
			Token:     "token-123",
			ExpiresAt: expires,
		},
	})
	model = updated.(*Model)

	if model.authFlowActive {
		t.Fatalf("expected auth flow to finish")
	}
	if model.authToken.Value != "token-123" {
		t.Fatalf("expected auth token to be applied")
	}
	if !model.client.HasHeader("Authorization") {
		t.Fatalf("expected authorization header to be set")
	}

	store := auth.NewStore(authPath)
	stored, err := store.Load()
	if err != nil {
		t.Fatalf("load auth token: %v", err)
	}
	if stored.Value != "token-123" {
		t.Fatalf("expected stored token, got %q", stored.Value)
	}
	if stored.DjangoURL != cfg.DjangoURL {
		t.Fatalf("expected stored django url, got %q", stored.DjangoURL)
	}
}

func TestAuthKeySignsOutClearsToken(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	t.Setenv("REPROQ_TUI_AUTH_FILE", authPath)

	cfg := config.DefaultConfig()
	cfg.WorkerMetricsURL = "http://worker.local/metrics"
	cfg.DjangoURL = "https://django.example.com"
	model := NewModel(cfg)

	expires := time.Now().Add(time.Hour)
	updated, _ := model.Update(authStatusMsg{
		status: auth.PairStatus{
			Status:    "approved",
			Token:     "token-123",
			ExpiresAt: expires,
		},
	})
	model = updated.(*Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	model = updated.(*Model)

	if model.authToken.Value != "" {
		t.Fatalf("expected auth token cleared")
	}
	if model.toast != "Signed out" {
		t.Fatalf("expected signed out toast, got %q", model.toast)
	}

	store := auth.NewStore(authPath)
	if _, err := store.Load(); err == nil {
		t.Fatalf("expected auth token to be cleared from store")
	}
}
