package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "reproq-tui",
	Short: "TUI dashboard for Reproq",
}

func Execute() error { return RootCmd.Execute() }

func ExitOnErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
