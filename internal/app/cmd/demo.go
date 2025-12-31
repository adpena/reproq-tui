package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/adpena/reproq-tui/internal/app/demo"
	"github.com/adpena/reproq-tui/internal/config"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run the dashboard with a local mock server",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := demo.Start()
		if err != nil {
			return err
		}
		defer server.Close(context.Background())

		cfg := config.DefaultConfig()
		cfg.WorkerMetricsURL = server.MetricsURL
		cfg.WorkerHealthURL = server.HealthURL
		cfg.EventsURL = server.EventsURL
		cfg.DjangoStatsURL = server.StatsURL
		cfg.Theme = "auto"
		cfg.Window = 5 * time.Minute

		fmt.Fprintf(cmd.OutOrStdout(), "Demo server running at %s\n", server.BaseURL)
		return runDashboard(cfg)
	},
}

func init() {
	RootCmd.AddCommand(demoCmd)
}
