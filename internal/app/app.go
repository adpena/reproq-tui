package app

import (
	"fmt"
	"os"

	"github.com/adpena/reproq-tui/internal/app/cmd"
)

func Run() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "dashboard")
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
