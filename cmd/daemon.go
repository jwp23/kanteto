package cmd

import (
	"context"
	"fmt"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the reminder daemon",
	Long:  "Start, stop, or check status of the background reminder daemon.",
}

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start the reminder daemon",
	Long:    "Run the background daemon that checks for due reminders every 30 seconds and plays an audible alert.",
	Example: `  kt daemon start`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return daemon.Run(context.Background(), svc, cfg)
	},
}

var stopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop the running daemon",
	Example: `  kt daemon stop`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.Stop(); err != nil {
			return err
		}
		fmt.Println("Daemon stopped.")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Check if the daemon is running",
	Example: `  kt daemon status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		running, pid, err := daemon.IsRunning()
		if err != nil {
			return err
		}
		if running {
			fmt.Printf("Daemon is running (pid %d)\n", pid)
		} else {
			fmt.Println("Daemon is not running.")
		}
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(startCmd)
	daemonCmd.AddCommand(stopCmd)
	daemonCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(daemonCmd)
}
