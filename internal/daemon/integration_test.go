//go:build integration

package daemon

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func integrationSvc(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return task.NewService(s)
}

func TestDaemon_ReminderFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	svc := integrationSvc(t)
	svc.SetLeadTime(0)
	player := &mockPlayer{}
	cfg := config.Config{SoundFile: "alert.wav"}

	// Add a task with RemindAt in the past
	past := time.Now().Add(-1 * time.Minute)
	tk, err := svc.Add("integration reminder", &past)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWithPlayer(ctx, svc, cfg, player)
	}()

	// Wait enough for the immediate check to fire
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not exit")
	}

	// Verify reminder was fired
	if len(player.calls) == 0 {
		t.Error("expected Play to be called for due reminder")
	}

	// Verify task was marked as reminded
	updated, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Reminded {
		t.Error("task should be marked as reminded")
	}
}

func TestDaemon_PIDCleanup(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	svc := integrationSvc(t)
	cfg := config.Config{}
	player := &mockPlayer{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWithPlayer(ctx, svc, cfg, player)
	}()

	// Wait for PID file to be written
	time.Sleep(100 * time.Millisecond)

	pidPath := PIDPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("PID file should exist while daemon is running")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not exit")
	}

	// PID file should be cleaned up
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after daemon stops")
	}
}

func TestDaemon_ConcurrentAccess(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Use a file-backed SQLite DB for WAL testing
	dbPath := tmp + "/kanteto/test.db"
	if err := os.MkdirAll(tmp+"/kanteto", 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := task.NewService(s)
	svc.SetLeadTime(0)

	player := &mockPlayer{}
	cfg := config.Config{SoundFile: "alert.wav"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunWithPlayer(ctx, svc, cfg, player)
	}()

	// Wait for daemon to start
	time.Sleep(200 * time.Millisecond)

	// Add a task from the "main" goroutine while daemon is running
	past := time.Now().Add(-1 * time.Minute)
	_, err = svc.Add("concurrent task", &past)
	if err != nil {
		t.Fatalf("concurrent Add failed (WAL issue?): %v", err)
	}

	// Verify the task was created
	tasks, err := svc.ListAll()
	if err != nil {
		t.Fatalf("concurrent ListAll failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	cancel()
	<-done
}
