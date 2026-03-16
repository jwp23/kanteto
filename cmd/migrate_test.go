package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"

	_ "modernc.org/sqlite"
)

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not found on PATH, skipping integration test")
	}
}

func createSQLiteDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY);
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY, title TEXT NOT NULL, due_at DATETIME,
		completed INTEGER NOT NULL DEFAULT 0, completed_at DATETIME,
		created_at DATETIME NOT NULL, recurrence_pattern TEXT,
		recurrence_time TEXT,
		tags TEXT NOT NULL DEFAULT '[]', profile TEXT NOT NULL DEFAULT 'default'
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func insertSQLiteTask(t *testing.T, db *sql.DB, tk task.Task) {
	t.Helper()
	var dueAt, completedAt any
	if tk.DueAt != nil {
		dueAt = *tk.DueAt
	}
	if tk.CompletedAt != nil {
		completedAt = *tk.CompletedAt
	}
	var recPat, recTime any
	if tk.RecurrencePattern != "" {
		recPat = tk.RecurrencePattern
	}
	if tk.RecurrenceTime != "" {
		recTime = tk.RecurrenceTime
	}
	tags := "[]"
	if len(tk.Tags) > 0 {
		data, _ := json.Marshal(tk.Tags)
		tags = string(data)
	}
	completed := 0
	if tk.Completed {
		completed = 1
	}
	_, err := db.Exec(`INSERT INTO tasks (id, title, due_at, completed, completed_at, created_at,
		recurrence_pattern, recurrence_time, tags, profile)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tk.ID, tk.Title, dueAt, completed, completedAt, tk.CreatedAt,
		recPat, recTime, tags, tk.Profile)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
}

func TestMigrate_HappyPath(t *testing.T) {
	skipIfNoDolt(t)

	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	dbPath := filepath.Join(dataDir, "kanteto.db")
	db := createSQLiteDB(t, dbPath)

	now := time.Now().Truncate(time.Second)
	due := now.Add(2 * time.Hour)

	tasks := []task.Task{
		{ID: task.NewID(), Title: "Simple task", CreatedAt: now, Tags: []string{}},
		{ID: task.NewID(), Title: "Due task", DueAt: &due, CreatedAt: now, Tags: []string{}},
		{ID: task.NewID(), Title: "Tagged task", CreatedAt: now, Tags: []string{"work", "urgent"}},
		{ID: task.NewID(), Title: "Profiled task", CreatedAt: now, Profile: "work", Tags: []string{}},
		{ID: task.NewID(), Title: "Recurring task", CreatedAt: now, DueAt: &due, RecurrencePattern: "daily", RecurrenceTime: "9:00", Tags: []string{}},
		{ID: task.NewID(), Title: "Done task", Completed: true, CompletedAt: &now, CreatedAt: now, Tags: []string{}},
	}

	for _, tk := range tasks {
		insertSQLiteTask(t, db, tk)
	}
	db.Close()

	t.Setenv("XDG_DATA_HOME", parentDir)
	out, err := execMigrate(t)
	if err != nil {
		t.Fatalf("migrate error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Migrated 6 tasks") {
		t.Errorf("expected 'Migrated 6 tasks', got:\n%s", out)
	}

	// Verify tasks in Dolt
	doltDir := filepath.Join(dataDir, "dolt")
	ds, err := store.New(doltDir)
	if err != nil {
		t.Fatalf("open dolt store: %v", err)
	}

	all, err := ds.ListAll(true)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 6 {
		t.Fatalf("expected 6 tasks in dolt, got %d", len(all))
	}

	for _, tk := range all {
		switch tk.Title {
		case "Tagged task":
			if len(tk.Tags) != 2 || tk.Tags[0] != "work" || tk.Tags[1] != "urgent" {
				t.Errorf("tags not preserved: %v", tk.Tags)
			}
		case "Profiled task":
			if tk.Profile != "work" {
				t.Errorf("profile not preserved: %q", tk.Profile)
			}
		case "Done task":
			if !tk.Completed {
				t.Error("completed status not preserved")
			}
		case "Recurring task":
			if tk.RecurrencePattern != "daily" || tk.RecurrenceTime != "9:00" {
				t.Errorf("recurrence not preserved: %q %q", tk.RecurrencePattern, tk.RecurrenceTime)
			}
		}
	}
}

func TestMigrate_NoSQLiteFile(t *testing.T) {
	skipIfNoDolt(t)
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	t.Setenv("XDG_DATA_HOME", parentDir)
	_, err := execMigrate(t)
	if err == nil {
		t.Error("expected error when no SQLite file exists")
	}
}

func TestMigrate_AlreadyMigrated(t *testing.T) {
	skipIfNoDolt(t)
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	// Create SQLite DB with a task
	dbPath := filepath.Join(dataDir, "kanteto.db")
	db := createSQLiteDB(t, dbPath)
	now := time.Now().Truncate(time.Second)
	insertSQLiteTask(t, db, task.Task{ID: task.NewID(), Title: "Original", CreatedAt: now, Tags: []string{}})
	db.Close()

	// Run first migration
	t.Setenv("XDG_DATA_HOME", parentDir)
	out, err := execMigrate(t)
	if err != nil {
		t.Fatalf("first migrate error: %v\noutput: %s", err, out)
	}

	// Re-running without --force should fail
	_, err = execMigrate(t)
	if err == nil {
		t.Error("expected error when tasks already exist in Dolt")
	}
	if err != nil && !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected error to mention --force, got: %v", err)
	}
}

func TestMigrate_ExistingEmptyRepo(t *testing.T) {
	skipIfNoDolt(t)
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	// Create SQLite DB with a task
	dbPath := filepath.Join(dataDir, "kanteto.db")
	db := createSQLiteDB(t, dbPath)
	now := time.Now().Truncate(time.Second)
	insertSQLiteTask(t, db, task.Task{ID: task.NewID(), Title: "Migrate me", CreatedAt: now, Tags: []string{}})
	db.Close()

	// Pre-init dolt repo (simulates user running `dolt init` manually)
	doltDir := filepath.Join(dataDir, "dolt")
	os.MkdirAll(doltDir, 0o755)
	initCmd := exec.Command("dolt", "init")
	initCmd.Dir = doltDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}

	// Migrate should succeed — store.New creates the tasks table idempotently
	t.Setenv("XDG_DATA_HOME", parentDir)
	out, err := execMigrate(t)
	if err != nil {
		t.Fatalf("migrate error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Migrated 1 tasks") {
		t.Errorf("expected 'Migrated 1 tasks', got:\n%s", out)
	}
}

func TestMigrate_OldSchema(t *testing.T) {
	skipIfNoDolt(t)
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	// Create SQLite DB with old schema (no tags, profile, recurrence columns)
	dbPath := filepath.Join(dataDir, "kanteto.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, due_at DATETIME,
			completed INTEGER NOT NULL DEFAULT 0, completed_at DATETIME,
			created_at DATETIME NOT NULL
		);`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	now := time.Now().Truncate(time.Second)
	_, err = db.Exec(`INSERT INTO tasks (id, title, completed, created_at) VALUES (?, ?, 0, ?)`,
		"old-1", "Old task", now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()

	t.Setenv("XDG_DATA_HOME", parentDir)
	out, migErr := execMigrate(t)
	if migErr != nil {
		t.Fatalf("migrate error: %v\noutput: %s", migErr, out)
	}
	if !strings.Contains(out, "Migrated 1 tasks") {
		t.Errorf("expected 'Migrated 1 tasks', got:\n%s", out)
	}

	// Verify defaults were applied
	doltDir := filepath.Join(dataDir, "dolt")
	ds, err := store.New(doltDir)
	if err != nil {
		t.Fatalf("open dolt: %v", err)
	}
	got, err := ds.Get("old-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", got.Tags)
	}
	if got.Profile != "default" {
		t.Errorf("expected profile 'default', got %q", got.Profile)
	}
}

func execMigrate(t *testing.T, extraArgs ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"migrate"}, extraArgs...))

	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = origStdout

	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)
	combined := buf.String() + stdoutBuf.String()
	return combined, err
}
