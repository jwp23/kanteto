package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/sync"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/jwp23/kanteto/internal/tui"
	"github.com/spf13/cobra"
)

var (
	svc             *task.Service
	syncer          *sync.Sync
	cfg             config.Config
	profileOverride string
)

var rootCmd = &cobra.Command{
	Use:     "kt",
	Short:   "Kanteto — track small tasks and promises",
	Long:    "A TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time.",
	Version: "0.4.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initService()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		player := tui.NewSoundPlayer(cfg.SoundFile)
		p := tui.New(svc, activeProfile(), syncer, player)
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
	doltDir := filepath.Join(dataDir, "dolt")
	if err := os.MkdirAll(doltDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	s, err := store.New(doltDir)
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
	syncer = sync.New(doltDir)
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileOverride, "profile", "", "override active profile for this command")
}
