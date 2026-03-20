package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	syncsvc "github.com/jwp23/kanteto/internal/sync"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/jwp23/kanteto/internal/tui"
	"github.com/spf13/cobra"

	_ "github.com/dolthub/driver"
)

var (
	svc             *task.Service
	syncer          *syncsvc.Sync
	cfg             config.Config
	profileOverride string
)

var rootCmd = &cobra.Command{
	Use:     "kt",
	Short:   "Kanteto — track small tasks and promises",
	Long:    "A TUI tool for tracking small tasks and promises that are too small for tickets but still need to get done on time.",
	Version: "0.5.0",
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

// openDoltDB opens a Dolt embedded database at the given data directory.
func openDoltDB(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := fmt.Sprintf("file://%s?commitname=kanteto&commitemail=kanteto@local&database=kanteto", dataDir)
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("open dolt: %w", err)
	}

	// Ensure the database exists (first-run scenario).
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS kanteto"); err != nil {
		db.Close()
		return nil, fmt.Errorf("create database: %w", err)
	}
	if _, err := db.Exec("USE kanteto"); err != nil {
		db.Close()
		return nil, fmt.Errorf("use database: %w", err)
	}
	return db, nil
}

func initService() error {
	dataDir := config.DataDir()
	db, err := openDoltDB(dataDir)
	if err != nil {
		return err
	}

	s, err := store.New(db)
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
	syncer = syncsvc.New(db)
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileOverride, "profile", "", "override active profile for this command")
}
