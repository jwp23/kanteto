package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/task"
)

const checkInterval = 30 * time.Second

// Run starts the daemon loop that checks for due reminders.
func Run(svc *task.Service, cfg config.Config) error {
	pidPath := filepath.Join(config.DataDir(), "daemon.pid")
	if err := writePID(pidPath); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	defer os.Remove(pidPath)

	log.Printf("kanteto daemon started (pid %d, checking every %s)", os.Getpid(), checkInterval)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Check immediately on start
	checkReminders(svc, cfg)

	for range ticker.C {
		checkReminders(svc, cfg)
	}

	return nil
}

func checkReminders(svc *task.Service, cfg config.Config) {
	tasks, err := svc.GetDueReminders()
	if err != nil {
		log.Printf("error checking reminders: %v", err)
		return
	}

	for _, t := range tasks {
		log.Printf("REMINDER: %s (due %s)", t.Title, t.DueAt.Format("3:04PM"))
		playSound(cfg.SoundFile)
		if err := svc.MarkReminded(t.ID); err != nil {
			log.Printf("error marking reminded: %v", err)
		}
	}
}

func playSound(soundFile string) {
	cmd := soundCommand(soundFile)
	if cmd == "" {
		// No sound player available, log only
		return
	}

	var c *exec.Cmd
	if soundFile != "" {
		c = exec.Command(cmd, soundFile)
	} else {
		// Try system bell as fallback
		fmt.Print("\a")
		return
	}

	if err := c.Start(); err != nil {
		log.Printf("error playing sound: %v", err)
	}
}

func soundCommand(soundFile string) string {
	if soundFile != "" {
		// Try platform-specific players
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
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}
