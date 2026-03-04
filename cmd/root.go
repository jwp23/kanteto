package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/jwp23/kanteto/internal/tui"
	"github.com/spf13/cobra"
)

var svc *task.Service

var rootCmd = &cobra.Command{
	Use:   "kt",
	Short: "Kanteto — track small tasks and promises",
	Long:  "A CLI and TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initService()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tui.New(svc)
		_, err := p.Run()
		return err
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func initService() error {
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "kanteto.db")
	s, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	svc = task.NewService(s)
	svc.SetLeadTime(cfg.ReminderLeadTime)
	return nil
}
