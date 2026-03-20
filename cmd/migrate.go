package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	syncsvc "github.com/jwp23/kanteto/internal/sync"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"

	_ "github.com/dolthub/driver"
	_ "modernc.org/sqlite"
)

var migrateForce bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate tasks from SQLite to Dolt",
	Long:  "One-time migration: reads all tasks from the SQLite database and writes them to a new Dolt repository.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil // migrate manages its own Dolt initialization
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		sqlitePath := filepath.Join(dataDir, "kanteto.db")

		// Check SQLite file exists
		if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
			return fmt.Errorf("no SQLite database found at %s", sqlitePath)
		}

		// Read all tasks from SQLite
		tasks, err := readSQLiteTasks(sqlitePath)
		if err != nil {
			return fmt.Errorf("read SQLite: %w", err)
		}

		// Open Dolt database
		db, err := openDoltDB(dataDir)
		if err != nil {
			return fmt.Errorf("open dolt: %w", err)
		}
		defer db.Close()

		doltStore, err := store.New(db)
		if err != nil {
			return fmt.Errorf("init dolt: %w", err)
		}

		// Check if tasks already exist in Dolt
		existing, err := doltStore.ListAll(true)
		if err != nil {
			return fmt.Errorf("check existing tasks: %w", err)
		}
		if len(existing) > 0 && !migrateForce {
			return fmt.Errorf("Dolt repo already contains %d tasks — use --force to re-migrate", len(existing))
		}

		// Write all tasks to Dolt
		for _, t := range tasks {
			if err := doltStore.Create(t); err != nil {
				return fmt.Errorf("write task %s: %w", t.ID, err)
			}
		}

		// Commit the migration
		s := syncsvc.New(db)
		if err := s.Commit("migrate: import from SQLite"); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		fmt.Printf("Migrated %d tasks from SQLite to Dolt.\n", len(tasks))
		fmt.Printf("Dolt repo: %s\n", filepath.Join(dataDir, "dolt"))
		fmt.Println("You can safely delete the old SQLite file after verifying the migration.")
		return nil
	},
}

// sqliteColumns returns the set of column names present in the tasks table.
func sqliteColumns(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// readSQLiteTasks opens a SQLite database directly and reads all tasks.
// Handles older schemas that may lack tags, profile, or recurrence columns.
func readSQLiteTasks(dsn string) ([]task.Task, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	cols, err := sqliteColumns(db)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	// Build SELECT with defaults for missing columns
	selects := []string{"id", "title", "due_at", "completed", "completed_at", "created_at"}
	if cols["recurrence_pattern"] {
		selects = append(selects, "recurrence_pattern")
	} else {
		selects = append(selects, "NULL AS recurrence_pattern")
	}
	if cols["recurrence_time"] {
		selects = append(selects, "recurrence_time")
	} else {
		selects = append(selects, "NULL AS recurrence_time")
	}
	if cols["tags"] {
		selects = append(selects, "tags")
	} else {
		selects = append(selects, "'[]' AS tags")
	}
	if cols["profile"] {
		selects = append(selects, "profile")
	} else {
		selects = append(selects, "'default' AS profile")
	}

	q := "SELECT " + strings.Join(selects, ", ") + " FROM tasks"
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var dueAt, completedAt sql.NullTime
		var recurrencePattern, recurrenceTime sql.NullString
		var completed int
		var tagsJSON string

		err := rows.Scan(
			&t.ID, &t.Title, &dueAt, &completed, &completedAt, &t.CreatedAt,
			&recurrencePattern, &recurrenceTime,
			&tagsJSON, &t.Profile,
		)
		if err != nil {
			return nil, err
		}

		t.Completed = completed != 0
		if dueAt.Valid {
			t.DueAt = &dueAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		if recurrencePattern.Valid {
			t.RecurrencePattern = recurrencePattern.String
		}
		if recurrenceTime.Valid {
			t.RecurrenceTime = recurrenceTime.String
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
	migrateCmd.Flags().BoolVar(&migrateForce, "force", false, "re-run migration even if tasks already exist in Dolt")
	rootCmd.AddCommand(migrateCmd)
}
