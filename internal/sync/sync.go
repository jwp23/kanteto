package sync

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Sync provides Dolt sync operations (push/pull/remote management).
type Sync struct {
	dir string
}

// New creates a Sync instance operating on the given Dolt repo directory.
func New(dir string) *Sync {
	return &Sync{dir: dir}
}

// Dir returns the Dolt repo directory.
func (s *Sync) Dir() string {
	return s.dir
}

// InitRepo initializes a new Dolt repo in the directory.
func (s *Sync) InitRepo() error {
	return s.runDolt("init")
}

// Push stages all changes, commits with a timestamp, and pushes to origin.
func (s *Sync) Push() error {
	if err := s.runDolt("add", "-A"); err != nil {
		return fmt.Errorf("stage: %w", err)
	}

	clean, err := s.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		msg := fmt.Sprintf("sync: %s", time.Now().Format("2006-01-02 15:04:05"))
		if err := s.runDolt("commit", "-m", msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	if err := s.runDolt("push", "origin", "main"); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

// Pull fetches and merges from origin.
func (s *Sync) Pull() error {
	if err := s.runDolt("pull", "origin"); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

// Commit stages all changes and creates a commit with the given message.
func (s *Sync) Commit(msg string) error {
	if err := s.runDolt("add", "-A"); err != nil {
		return fmt.Errorf("stage: %w", err)
	}
	if err := s.runDolt("commit", "-m", msg); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// AddRemote registers a new remote.
func (s *Sync) AddRemote(name, url string) error {
	return s.runDolt("remote", "add", name, url)
}

// ListRemotes returns configured remotes (one per line, "name url" format).
func (s *Sync) ListRemotes() ([]string, error) {
	out, err := s.runDoltOutput("remote", "-v")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	var remotes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

// HasRemote checks if a named remote exists.
func (s *Sync) HasRemote(name string) bool {
	remotes, err := s.ListRemotes()
	if err != nil {
		return false
	}
	for _, r := range remotes {
		if strings.HasPrefix(r, name+"\t") || strings.HasPrefix(r, name+" ") {
			return true
		}
	}
	return false
}

// IsClean returns true if the working set has no uncommitted changes.
func (s *Sync) IsClean() (bool, error) {
	out, err := s.runDoltOutput("status")
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "nothing to commit"), nil
}

func (s *Sync) runDolt(args ...string) error {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dolt %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (s *Sync) runDoltOutput(args ...string) (string, error) {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dolt %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}
