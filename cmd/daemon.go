package cmd

import (
	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the reminder daemon",
	Long:  "Run the background daemon that checks for due reminders every 30 seconds and plays an audible alert.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return daemon.Run(svc, cfg)
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
