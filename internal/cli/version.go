package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information - updated with each release
const (
	Version     = "1.3.1"
	ReleaseDate = "2026-01-15"
	Features    = "Fixed auto-retry detection for new rate limit message format"
)

func newVersionCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the current version of bmad-automate and build information.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "bmad-automate version %s\n", Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Release date: %s\n", ReleaseDate)
			fmt.Fprintf(cmd.OutOrStdout(), "Features: %s\n", Features)
		},
	}
}
