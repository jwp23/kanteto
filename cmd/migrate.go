package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/store/doltstore"
	syncsvc "github.com/jwp23/kanteto/internal/sync"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate tasks from SQLite to Dolt",
	Long:  "One-time migration: reads all tasks from the SQLite database and writes them to a new Dolt repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		sqlitePath := filepath.Join(dataDir, "kanteto.db")
		doltDir := filepath.Join(dataDir, "dolt")

		// Check SQLite file exists
		if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
			return fmt.Errorf("no SQLite database found at %s", sqlitePath)
		}

		// Check Dolt repo doesn't already exist
		if _, err := os.Stat(filepath.Join(doltDir, ".dolt")); err == nil {
			return fmt.Errorf("Dolt repo already exists at %s — migration already done?", doltDir)
		}

		// Open SQLite store
		sqliteStore, err := store.New(sqlitePath)
		if err != nil {
			return fmt.Errorf("open SQLite: %w", err)
		}
		defer sqliteStore.Close()

		// Read all tasks (including completed)
		tasks, err := sqliteStore.ListAll(true)
		if err != nil {
			return fmt.Errorf("read tasks: %w", err)
		}

		// Create Dolt repo
		if err := os.MkdirAll(doltDir, 0o755); err != nil {
			return fmt.Errorf("create dolt dir: %w", err)
		}
		doltStore, err := doltstore.New(doltDir)
		if err != nil {
			return fmt.Errorf("init dolt: %w", err)
		}

		// Write all tasks to Dolt
		for _, t := range tasks {
			if err := doltStore.Create(t); err != nil {
				return fmt.Errorf("write task %s: %w", t.ID, err)
			}
		}

		// Commit the migration
		s := syncsvc.New(doltDir)
		if err := s.Commit("migrate: import from SQLite"); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		fmt.Printf("Migrated %d tasks from SQLite to Dolt.\n", len(tasks))
		fmt.Printf("Dolt repo: %s\n", doltDir)
		fmt.Println("You can safely delete the old SQLite file after verifying the migration.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
