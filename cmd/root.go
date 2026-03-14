package cmd

import (
	"fmt"
	"os"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/jwp23/kanteto/internal/tui"
	"github.com/spf13/cobra"
)

var (
	svc             *task.Service
	cfg             config.Config
	profileOverride string
)

var rootCmd = &cobra.Command{
	Use:     "kt",
	Short:   "Kanteto — track small tasks and promises",
	Long:    "A TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time.",
	Version: "0.2.6",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initService()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tui.New(svc, activeProfile())
		_, err := p.Run()
		return err
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func activeProfile() string {
	if profileOverride != "" {
		return profileOverride
	}
	return cfg.ActiveProfile
}

func initService() error {
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	s, err := store.New(dataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	var loadErr error
	cfg, loadErr = config.Load()
	if loadErr != nil {
		return fmt.Errorf("load config: %w", loadErr)
	}

	profile := activeProfile()
	var repo task.Repository = s
	if profile != "" {
		repo = store.NewProfileStore(s, profile)
	}

	svc = task.NewService(repo)
	svc.SetLeadTime(cfg.ReminderLeadTime)
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileOverride, "profile", "", "override active profile for this command")
}
