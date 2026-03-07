package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

// mockPlayer records Play calls for testing.
type mockPlayer struct {
	calls []string
}

func (p *mockPlayer) Play(soundFile string) {
	p.calls = append(p.calls, soundFile)
}

func testSvc(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return task.NewService(s)
}

func TestSoundCommand(t *testing.T) {
	cmd := soundCommand("")
	if cmd == "" {
		t.Skip("no sound player found on this system")
	}
	if cmd != "paplay" && cmd != "afplay" && cmd != "aplay" {
		t.Errorf("unexpected sound command: %q", cmd)
	}
}

func TestCheckInterval(t *testing.T) {
	if checkInterval <= 0 {
		t.Error("checkInterval must be positive")
	}
	if checkInterval > time.Minute {
		t.Error("checkInterval should be <= 1 minute for timely reminders")
	}
}

func TestPIDPath(t *testing.T) {
	p := PIDPath()
	if p == "" {
		t.Error("PIDPath should not be empty")
	}
}

func TestIsRunning_NoPIDFile(t *testing.T) {
	// Use a temp dir so there's no PID file
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	running, _, err := IsRunning()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("should not be running when no PID file exists")
	}
}

func TestIsRunning_ValidPID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Write our own PID — we know this process is alive
	pidPath := PIDPath()
	if err := writePID(pidPath); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pidPath)

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !running {
		t.Error("should report running for current process PID")
	}
	if pid != os.Getpid() {
		t.Errorf("expected pid %d, got %d", os.Getpid(), pid)
	}
}

func TestIsRunning_StalePID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Write a PID that is almost certainly not running
	pidPath := PIDPath()
	if err := os.MkdirAll(tmp+"/kanteto", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(999999)), 0o600); err != nil {
		t.Fatal(err)
	}

	running, _, err := IsRunning()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("stale PID should not be reported as running")
	}

	// PID file should be cleaned up
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("stale PID file should be removed")
	}
}

func TestStop_NotRunning(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	err := Stop()
	if err == nil {
		t.Error("expected error when daemon is not running")
	}
}

func TestStop_RunningProcess(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Spawn a subprocess we can safely terminate
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Write the subprocess PID
	pidPath := PIDPath()
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// PID file should be removed
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after Stop")
	}
}

func TestRun_DuplicatePrevention(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Write our own PID to simulate a running daemon
	pidPath := PIDPath()
	if err := writePID(pidPath); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pidPath)

	svc := testSvc(t)
	cfg := config.Config{}
	err := Run(context.Background(), svc, cfg)
	if err == nil {
		t.Error("expected error when daemon is already running")
	}
}

func TestCheckReminders_FiresAndMarks(t *testing.T) {
	svc := testSvc(t)
	svc.SetLeadTime(0) // RemindAt == DueAt
	player := &mockPlayer{}

	// Create a task due in the past so RemindAt is also in the past
	past := time.Now().Add(-5 * time.Minute)
	if _, err := svc.Add("test reminder", &past); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{SoundFile: "test.wav"}
	checkReminders(svc, cfg, player)

	if len(player.calls) == 0 {
		t.Error("expected player.Play to be called")
	}
	if len(player.calls) > 0 && player.calls[0] != "test.wav" {
		t.Errorf("expected Play called with 'test.wav', got %q", player.calls[0])
	}
}

func TestCheckReminders_AlreadyReminded(t *testing.T) {
	svc := testSvc(t)
	player := &mockPlayer{}

	// Add a task due in the past
	past := time.Now().Add(-5 * time.Minute)
	svc.SetLeadTime(0) // No lead time so RemindAt == DueAt
	tk, err := svc.Add("reminded task", &past)
	if err != nil {
		t.Fatal(err)
	}

	// Mark it as already reminded
	if err := svc.MarkReminded(tk.ID); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{SoundFile: "test.wav"}
	checkReminders(svc, cfg, player)

	if len(player.calls) != 0 {
		t.Error("should not fire for already-reminded task")
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	svc := testSvc(t)
	cfg := config.Config{}
	player := &mockPlayer{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWithPlayer(ctx, svc, cfg, player)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil error on context cancellation, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}

	// PID file should be cleaned up
	pidPath := PIDPath()
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after clean exit")
	}
}
