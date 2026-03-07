package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/task"
)

const checkInterval = 30 * time.Second

// SoundPlayer plays reminder sounds.
type SoundPlayer interface {
	Play(soundFile string)
}

// defaultPlayer uses OS sound commands.
type defaultPlayer struct{}

func (defaultPlayer) Play(soundFile string) {
	playSound(soundFile)
}

// PIDPath returns the path to the daemon PID file.
func PIDPath() string {
	return filepath.Join(config.DataDir(), "daemon.pid")
}

// IsRunning checks if a daemon process is already running.
// Returns running status, PID, and any error.
func IsRunning() (bool, int, error) {
	data, err := os.ReadFile(PIDPath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupted PID file, clean up
		os.Remove(PIDPath())
		return false, 0, nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(PIDPath())
		return false, 0, nil
	}

	// Signal 0 checks if process exists without sending a signal
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		// Process not running, stale PID file
		os.Remove(PIDPath())
		return false, pid, nil
	}

	return true, pid, nil
}

// Stop sends SIGTERM to the running daemon and removes the PID file.
func Stop() error {
	running, pid, err := IsRunning()
	if err != nil {
		return fmt.Errorf("check daemon status: %w", err)
	}
	if !running {
		return fmt.Errorf("daemon is not running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to %d: %w", pid, err)
	}

	os.Remove(PIDPath())
	return nil
}

// Run starts the daemon loop that checks for due reminders.
func Run(ctx context.Context, svc *task.Service, cfg config.Config) error {
	return RunWithPlayer(ctx, svc, cfg, defaultPlayer{})
}

// RunWithPlayer starts the daemon loop using the given SoundPlayer.
func RunWithPlayer(ctx context.Context, svc *task.Service, cfg config.Config, player SoundPlayer) error {
	// Check for duplicate instance
	running, pid, err := IsRunning()
	if err != nil {
		return fmt.Errorf("check existing daemon: %w", err)
	}
	if running {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	pidPath := PIDPath()
	if err := writePID(pidPath); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	defer os.Remove(pidPath)

	// Handle OS signals for clean shutdown
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("kanteto daemon started (pid %d, checking every %s)", os.Getpid(), checkInterval)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Check immediately on start
	checkReminders(svc, cfg, player)

	for {
		select {
		case <-ctx.Done():
			log.Printf("kanteto daemon stopped")
			return nil
		case <-ticker.C:
			checkReminders(svc, cfg, player)
		}
	}
}

func checkReminders(svc *task.Service, cfg config.Config, player SoundPlayer) {
	tasks, err := svc.GetDueReminders()
	if err != nil {
		log.Printf("error checking reminders: %v", err)
		return
	}

	for _, t := range tasks {
		log.Printf("REMINDER: %s (due %s)", t.Title, t.DueAt.Format("3:04PM"))
		player.Play(cfg.SoundFile)
		if err := svc.MarkReminded(t.ID); err != nil {
			log.Printf("error marking reminded: %v", err)
		}
	}
}

func playSound(soundFile string) {
	cmd := soundCommand(soundFile)
	if cmd == "" {
		return
	}

	var c *exec.Cmd
	if soundFile != "" {
		c = exec.Command(cmd, soundFile)
	} else {
		fmt.Print("\a")
		return
	}

	if err := c.Start(); err != nil {
		log.Printf("error playing sound: %v", err)
	}
}

func soundCommand(soundFile string) string {
	if soundFile != "" {
		for _, player := range []string{"paplay", "afplay", "aplay"} {
			if _, err := exec.LookPath(player); err == nil {
				return player
			}
		}
	}
	return ""
}

func writePID(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600)
}
