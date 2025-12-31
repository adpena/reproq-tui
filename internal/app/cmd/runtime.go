package cmd

import (
	"github.com/adpena/reproq-tui/internal/config"
	"github.com/adpena/reproq-tui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func runDashboard(cfg config.Config) error {
	model := ui.NewModel(cfg)
	defer model.Close()
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}
