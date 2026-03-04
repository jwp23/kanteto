package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "kt",
	Short: "Kanteto — track small tasks and promises",
	Long:  "A CLI and TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
