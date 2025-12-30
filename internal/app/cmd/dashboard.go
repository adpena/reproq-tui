package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m model) View() string { return "dashboard: scaffold (press q)\n" }

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Run the dashboard UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(model{}, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func init() {
	RootCmd.AddCommand(dashboardCmd)
	fmt.Print("")
}
