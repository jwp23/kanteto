package sync

import (
	"database/sql"
	"fmt"
	"time"
)

// Sync provides Dolt sync operations via the embedded driver's DOLT_* procedures.
type Sync struct {
	db *sql.DB
}

// New creates a Sync instance using the provided *sql.DB.
func New(db *sql.DB) *Sync {
	return &Sync{db: db}
}

// Snapshot stages all changes and commits with the given message (no push).
func (s *Sync) Snapshot(msg string) error {
	if _, err := s.db.Exec("CALL DOLT_ADD('-A')"); err != nil {
		return fmt.Errorf("stage: %w", err)
	}

	clean, err := s.IsClean()
	if err != nil {
		return err
	}
	if clean {
		return nil
	}

	if _, err := s.db.Exec("CALL DOLT_COMMIT('-m', ?)", msg); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// PushRemote pushes to origin (no add/commit). For async background use.
func (s *Sync) PushRemote() error {
	if _, err := s.db.Exec("CALL DOLT_PUSH('origin', 'main')"); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

// Push stages all changes, commits with a timestamp, and pushes to origin.
func (s *Sync) Push() error {
	msg := fmt.Sprintf("sync: %s", time.Now().Format("2006-01-02 15:04:05"))
	if err := s.Snapshot(msg); err != nil {
		return err
	}
	return s.PushRemote()
}

// Pull fetches and merges from origin.
func (s *Sync) Pull() error {
	if _, err := s.db.Exec("CALL DOLT_PULL('origin')"); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

// Commit stages all changes and creates a commit with the given message.
func (s *Sync) Commit(msg string) error {
	return s.Snapshot(msg)
}

// AddRemote registers a new remote.
func (s *Sync) AddRemote(name, url string) error {
	_, err := s.db.Exec("CALL DOLT_REMOTE('add', ?, ?)", name, url)
	return err
}

// ListRemotes returns configured remotes as "name\turl" strings.
func (s *Sync) ListRemotes() ([]string, error) {
	rows, err := s.db.Query("SELECT name, url FROM dolt_remotes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var remotes []string
	for rows.Next() {
		var name, url string
		if err := rows.Scan(&name, &url); err != nil {
			return nil, err
		}
		remotes = append(remotes, name+"\t"+url)
	}
	return remotes, rows.Err()
}

// HasRemote checks if a named remote exists.
func (s *Sync) HasRemote(name string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM dolt_remotes WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// IsClean returns true if the working set has no uncommitted changes.
func (s *Sync) IsClean() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM dolt_status").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// InitRepo creates an initial commit in a new Dolt database.
func (s *Sync) InitRepo() error {
	if _, err := s.db.Exec("CALL DOLT_ADD('-A')"); err != nil {
		return fmt.Errorf("stage: %w", err)
	}
	if _, err := s.db.Exec("CALL DOLT_COMMIT('--allow-empty', '-m', 'init')"); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
