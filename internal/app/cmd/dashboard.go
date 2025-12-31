package cmd

import (
	"fmt"

	"github.com/adpena/reproq-tui/internal/config"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Run the dashboard UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadAllowEmpty(cmd)
		if err != nil {
			return err
		}
		closeLog, err := setupLogging(cfg.LogFile)
		if err != nil {
			return err
		}
		defer closeLog()
		return runDashboard(cfg)
	},
}

func init() {
	config.RegisterFlags(dashboardCmd)
	RootCmd.AddCommand(dashboardCmd)
	fmt.Print("")
}
