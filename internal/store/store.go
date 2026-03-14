package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwp23/kanteto/internal/task"

	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed task repository.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database and runs migrations.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{1, `ALTER TABLE tasks ADD COLUMN tags TEXT NOT NULL DEFAULT '[]'`},
	{2, `ALTER TABLE tasks ADD COLUMN profile TEXT NOT NULL DEFAULT 'default'`},
	{3, `CREATE INDEX IF NOT EXISTS idx_tasks_profile ON tasks(profile)`},
}

func (s *Store) migrate() error {
	baseSchema := `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	);
	CREATE TABLE IF NOT EXISTS tasks (
		id                  TEXT PRIMARY KEY,
		title               TEXT NOT NULL,
		due_at              DATETIME,
		completed           INTEGER NOT NULL DEFAULT 0,
		completed_at        DATETIME,
		created_at          DATETIME NOT NULL,
		remind_at           DATETIME,
		reminded            INTEGER NOT NULL DEFAULT 0,
		recurrence_pattern  TEXT,
		recurrence_time     TEXT,
		recurrence_next_due DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_due_at ON tasks(due_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_remind_at ON tasks(remind_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(completed);
	`
	if _, err := s.db.Exec(baseSchema); err != nil {
		return err
	}

	var version int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version > version {
			if _, err := s.db.Exec(m.sql); err != nil {
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
			if _, err := s.db.Exec("INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return fmt.Errorf("record migration v%d: %w", m.version, err)
			}
		}
	}

	return nil
}

// Create inserts a new task.
func (s *Store) Create(t task.Task) error {
	_, err := s.db.Exec(
		`INSERT INTO tasks (id, title, due_at, completed, created_at, remind_at, recurrence_pattern, recurrence_time, recurrence_next_due, tags, profile)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, timePtr(t.DueAt), boolToInt(t.Completed), t.CreatedAt,
		timePtr(t.RemindAt), nullStr(t.RecurrencePattern), nullStr(t.RecurrenceTime), timePtr(t.RecurrenceNextDue),
		marshalTags(t.Tags), t.Profile,
	)
	return err
}

const taskColumns = `id, title, due_at, completed, completed_at, created_at, remind_at, reminded,
	recurrence_pattern, recurrence_time, recurrence_next_due, tags, profile`

// Get retrieves a task by ID.
func (s *Store) Get(id string) (task.Task, error) {
	row := s.db.QueryRow(
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id,
	)
	return scanTask(row)
}

// Complete marks a task as completed.
func (s *Store) Complete(id string) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE tasks SET completed = 1, completed_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// Delete removes a task permanently.
func (s *Store) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// UpdateDueAt changes a task's due date (for snooze).
func (s *Store) UpdateDueAt(id string, dueAt *time.Time) error {
	_, err := s.db.Exec(`UPDATE tasks SET due_at = ?, reminded = 0 WHERE id = ?`, timePtr(dueAt), id)
	return err
}

// ListByDateRange returns incomplete tasks with due dates in [start, end).
func (s *Store) ListByDateRange(start, end time.Time) ([]task.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE completed = 0 AND due_at >= ? AND due_at < ?
		 ORDER BY due_at ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListAll returns all tasks, optionally including completed.
func (s *Store) ListAll(includeCompleted bool) ([]task.Task, error) {
	q := `SELECT ` + taskColumns + ` FROM tasks`
	if !includeCompleted {
		q += " WHERE completed = 0"
	}
	q += " ORDER BY due_at ASC NULLS LAST"

	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListUndated returns incomplete tasks with no due date.
func (s *Store) ListUndated() ([]task.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE completed = 0 AND due_at IS NULL
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListOverdue returns incomplete tasks with due dates before now.
func (s *Store) ListOverdue() ([]task.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE completed = 0 AND due_at < ?
		 ORDER BY due_at ASC`, time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListOverdueAsOf returns incomplete tasks with due dates before the given time.
func (s *Store) ListOverdueAsOf(asOf time.Time) ([]task.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE completed = 0 AND due_at < ?
		 ORDER BY due_at ASC`, asOf,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListDueReminders returns tasks that need reminders fired.
func (s *Store) ListDueReminders() ([]task.Task, error) {
	rows, err := s.db.Query(
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE completed = 0 AND reminded = 0 AND remind_at IS NOT NULL AND remind_at <= ?
		 ORDER BY remind_at ASC`, time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// MarkReminded sets the reminded flag on a task.
func (s *Store) MarkReminded(id string) error {
	_, err := s.db.Exec(`UPDATE tasks SET reminded = 1 WHERE id = ?`, id)
	return err
}

// Update saves changes to a task's recurrence and due date fields.
func (s *Store) Update(t task.Task) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET title = ?, due_at = ?, remind_at = ?, recurrence_pattern = ?,
		 recurrence_time = ?, recurrence_next_due = ?, completed = ?, completed_at = ?, reminded = ?,
		 tags = ?, profile = ?
		 WHERE id = ?`,
		t.Title, timePtr(t.DueAt), timePtr(t.RemindAt), nullStr(t.RecurrencePattern),
		nullStr(t.RecurrenceTime), timePtr(t.RecurrenceNextDue),
		boolToInt(t.Completed), timePtr(t.CompletedAt), boolToInt(t.Reminded),
		marshalTags(t.Tags), t.Profile, t.ID,
	)
	return err
}

// ListProfiles returns distinct profile names from all tasks.
func (s *Store) ListProfiles() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT profile FROM tasks ORDER BY profile`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// scanner is implemented by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (task.Task, error) {
	var t task.Task
	var dueAt, completedAt, remindAt, recurrenceNextDue sql.NullTime
	var recurrencePattern, recurrenceTime sql.NullString
	var completed, reminded int
	var tagsJSON string

	err := s.Scan(
		&t.ID, &t.Title, &dueAt, &completed, &completedAt, &t.CreatedAt,
		&remindAt, &reminded, &recurrencePattern, &recurrenceTime, &recurrenceNextDue,
		&tagsJSON, &t.Profile,
	)
	if err != nil {
		return t, err
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
	return t, nil
}

func scanTasks(rows *sql.Rows) ([]task.Task, error) {
	var tasks []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func marshalTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(tags)
	return string(data)
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
