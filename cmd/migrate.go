package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	syncsvc "github.com/jwp23/kanteto/internal/sync"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
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

		// Read all tasks from SQLite
		tasks, err := readSQLiteTasks(sqlitePath)
		if err != nil {
			return fmt.Errorf("read SQLite: %w", err)
		}

		// Create Dolt repo
		if err := os.MkdirAll(doltDir, 0o755); err != nil {
			return fmt.Errorf("create dolt dir: %w", err)
		}
		doltStore, err := store.New(doltDir)
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

// readSQLiteTasks opens a SQLite database directly and reads all tasks.
func readSQLiteTasks(dsn string) ([]task.Task, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, title, due_at, completed, completed_at, created_at,
		remind_at, reminded, recurrence_pattern, recurrence_time, recurrence_next_due,
		tags, profile FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var dueAt, completedAt, remindAt, recurrenceNextDue sql.NullTime
		var recurrencePattern, recurrenceTime sql.NullString
		var completed, reminded int
		var tagsJSON string

		err := rows.Scan(
			&t.ID, &t.Title, &dueAt, &completed, &completedAt, &t.CreatedAt,
			&remindAt, &reminded, &recurrencePattern, &recurrenceTime, &recurrenceNextDue,
			&tagsJSON, &t.Profile,
		)
		if err != nil {
			return nil, err
		}

		t.Completed = completed != 0
		t.Reminded = reminded != 0
		if dueAt.Valid {
			t.DueAt = &dueAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		if remindAt.Valid {
			t.RemindAt = &remindAt.Time
		}
		if recurrencePattern.Valid {
			t.RecurrencePattern = recurrencePattern.String
		}
		if recurrenceTime.Valid {
			t.RecurrenceTime = recurrenceTime.String
		}
		if recurrenceNextDue.Valid {
			t.RecurrenceNextDue = &recurrenceNextDue.Time
		}
		t.Tags = unmarshalTags(tagsJSON)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func unmarshalTags(s string) []string {
	var tags []string
	if s == "" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(s), &tags); err != nil {
		return []string{}
	}
	if tags == nil {
		return []string{}
	}
	return tags
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
