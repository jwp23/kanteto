package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/store/doltstore"
	"github.com/jwp23/kanteto/internal/task"
)

func TestMigrate_HappyPath(t *testing.T) {
	skipIfNoDolt(t)

	// Create a temp directory structure
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	// Create SQLite store with some tasks
	dbPath := filepath.Join(dataDir, "kanteto.db")
	sqliteStore, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create SQLite store: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	due := now.Add(2 * time.Hour)
	remind := now.Add(1 * time.Hour)

	tasks := []task.Task{
		{ID: task.NewID(), Title: "Simple task", CreatedAt: now},
		{ID: task.NewID(), Title: "Due task", DueAt: &due, RemindAt: &remind, CreatedAt: now},
		{ID: task.NewID(), Title: "Tagged task", CreatedAt: now, Tags: []string{"work", "urgent"}},
		{ID: task.NewID(), Title: "Profiled task", CreatedAt: now, Profile: "work"},
		{ID: task.NewID(), Title: "Recurring task", CreatedAt: now, DueAt: &due, RecurrencePattern: "daily", RecurrenceTime: "9:00"},
	}
	for _, tk := range tasks {
		if err := sqliteStore.Create(tk); err != nil {
			t.Fatalf("create task: %v", err)
		}
	}
	// Complete one task
	completedID := task.NewID()
	sqliteStore.Create(task.Task{ID: completedID, Title: "Done task", CreatedAt: now})
	sqliteStore.Complete(completedID)
	sqliteStore.Close()

	// Run migration
	t.Setenv("XDG_DATA_HOME", parentDir)
	out, err := execMigrate(t, parentDir)
	if err != nil {
		t.Fatalf("migrate error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Migrated 6 tasks") {
		t.Errorf("expected 'Migrated 6 tasks', got:\n%s", out)
	}

	// Verify tasks in Dolt
	doltDir := filepath.Join(dataDir, "dolt")
	ds, err := doltstore.New(doltDir)
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

	// Verify specific fields round-trip
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
	_, err := execMigrate(t, parentDir)
	if err == nil {
		t.Error("expected error when no SQLite file exists")
	}
}

func TestMigrate_AlreadyMigrated(t *testing.T) {
	skipIfNoDolt(t)
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "kanteto")
	os.MkdirAll(dataDir, 0o755)

	// Create SQLite store
	dbPath := filepath.Join(dataDir, "kanteto.db")
	sqliteStore, _ := store.New(dbPath)
	sqliteStore.Create(task.Task{ID: task.NewID(), Title: "Test", CreatedAt: time.Now()})
	sqliteStore.Close()

	// Pre-create dolt dir
	doltDir := filepath.Join(dataDir, "dolt")
	os.MkdirAll(doltDir, 0o755)
	cmd := exec.Command("dolt", "init")
	cmd.Dir = doltDir
	cmd.CombinedOutput()

	t.Setenv("XDG_DATA_HOME", parentDir)
	_, err := execMigrate(t, parentDir)
	if err == nil {
		t.Error("expected error when Dolt repo already exists")
	}
}

func execMigrate(t *testing.T, dataParentDir string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"migrate"})
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

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
