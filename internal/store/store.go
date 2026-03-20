package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

const timeLayout = "2006-01-02 15:04:05"

const taskColumns = `id, title, due_at, completed, completed_at, created_at,
	recurrence_pattern, recurrence_time, tags, profile`

// Store is a Dolt-backed task repository using the embedded driver.
type Store struct {
	db *sql.DB
}

// New creates a Store using the provided *sql.DB and ensures the schema exists.
func New(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}
	return s, nil
}

func (s *Store) ensureSchema() error {
	schema := `CREATE TABLE IF NOT EXISTS tasks (
		id                  VARCHAR(255) PRIMARY KEY,
		title               VARCHAR(1024) NOT NULL,
		due_at              DATETIME,
		completed           TINYINT(1) NOT NULL DEFAULT 0,
		completed_at        DATETIME,
		created_at          DATETIME NOT NULL,
		recurrence_pattern  VARCHAR(255),
		recurrence_time     VARCHAR(255),
		tags                JSON NOT NULL,
		profile             VARCHAR(255) NOT NULL DEFAULT 'default'
	);`
	_, err := s.db.Exec(schema)
	return err
}

// Close is a no-op; the caller owns the *sql.DB lifecycle.
func (s *Store) Close() error {
	return nil
}

// Create inserts a new task.
func (s *Store) Create(t task.Task) error {
	q := fmt.Sprintf(
		`INSERT INTO tasks (id, title, due_at, completed, created_at, recurrence_pattern, recurrence_time, tags, profile)
		 VALUES (%s, %s, %s, %d, %s, %s, %s, %s, %s)`,
		quote(t.ID), quote(t.Title), quoteTimePtr(t.DueAt), boolToInt(t.Completed), quoteTime(t.CreatedAt),
		quoteNullStr(t.RecurrencePattern), quoteNullStr(t.RecurrenceTime),
		quote(marshalTags(t.Tags)), quote(t.Profile),
	)
	_, err := s.db.Exec(q)
	return err
}

// Get retrieves a task by ID.
func (s *Store) Get(id string) (task.Task, error) {
	q := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = %s`, taskColumns, quote(id))
	rows, err := s.queryTasks(q)
	if err != nil {
		return task.Task{}, err
	}
	if len(rows) == 0 {
		return task.Task{}, fmt.Errorf("task not found: %s", id)
	}
	return rows[0], nil
}

// Complete marks a task as completed.
func (s *Store) Complete(id string) error {
	now := time.Now().Truncate(time.Second)
	q := fmt.Sprintf(
		`UPDATE tasks SET completed = 1, completed_at = %s WHERE id = %s`,
		quoteTime(now), quote(id),
	)
	_, err := s.db.Exec(q)
	return err
}

// Delete removes a task permanently.
func (s *Store) Delete(id string) error {
	q := fmt.Sprintf(`DELETE FROM tasks WHERE id = %s`, quote(id))
	_, err := s.db.Exec(q)
	return err
}

// Update saves changes to a task.
func (s *Store) Update(t task.Task) error {
	q := fmt.Sprintf(
		`UPDATE tasks SET title = %s, due_at = %s, recurrence_pattern = %s,
		 recurrence_time = %s, completed = %d, completed_at = %s,
		 tags = %s, profile = %s
		 WHERE id = %s`,
		quote(t.Title), quoteTimePtr(t.DueAt), quoteNullStr(t.RecurrencePattern),
		quoteNullStr(t.RecurrenceTime),
		boolToInt(t.Completed), quoteTimePtr(t.CompletedAt),
		quote(marshalTags(t.Tags)), quote(t.Profile), quote(t.ID),
	)
	_, err := s.db.Exec(q)
	return err
}

// UpdateDueAt changes a task's due date (for snooze).
func (s *Store) UpdateDueAt(id string, dueAt *time.Time) error {
	q := fmt.Sprintf(`UPDATE tasks SET due_at = %s WHERE id = %s`, quoteTimePtr(dueAt), quote(id))
	_, err := s.db.Exec(q)
	return err
}

// ListAll returns all tasks, optionally including completed.
func (s *Store) ListAll(includeCompleted bool) ([]task.Task, error) {
	q := "SELECT " + taskColumns + " FROM tasks"
	if !includeCompleted {
		q += " WHERE completed = 0"
	}
	q += " ORDER BY due_at IS NULL, due_at ASC"
	return s.queryTasks(q)
}

// ListByDateRange returns incomplete tasks with due dates in [start, end).
func (s *Store) ListByDateRange(start, end time.Time) ([]task.Task, error) {
	q := fmt.Sprintf(
		`SELECT %s FROM tasks WHERE completed = 0 AND due_at >= %s AND due_at < %s ORDER BY due_at ASC`,
		taskColumns, quoteTime(start), quoteTime(end),
	)
	return s.queryTasks(q)
}

// ListOverdue returns incomplete tasks with due dates before now.
func (s *Store) ListOverdue() ([]task.Task, error) {
	q := fmt.Sprintf(
		`SELECT %s FROM tasks WHERE completed = 0 AND due_at < %s ORDER BY due_at ASC`,
		taskColumns, quoteTime(time.Now()),
	)
	return s.queryTasks(q)
}

// ListOverdueAsOf returns incomplete tasks with due dates before the given time.
func (s *Store) ListOverdueAsOf(asOf time.Time) ([]task.Task, error) {
	q := fmt.Sprintf(
		`SELECT %s FROM tasks WHERE completed = 0 AND due_at < %s ORDER BY due_at ASC`,
		taskColumns, quoteTime(asOf),
	)
	return s.queryTasks(q)
}

// ListUndated returns incomplete tasks with no due date.
func (s *Store) ListUndated() ([]task.Task, error) {
	q := fmt.Sprintf(
		`SELECT %s FROM tasks WHERE completed = 0 AND due_at IS NULL ORDER BY created_at ASC`,
		taskColumns,
	)
	return s.queryTasks(q)
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

// queryTasks runs a SELECT query and scans results into tasks.
func (s *Store) queryTasks(q string) ([]task.Task, error) {
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var dueAt, completedAt sql.NullTime
		var createdAt time.Time
		var completed bool
		var recurrencePattern, recurrenceTime sql.NullString
		var tagsJSON string

		err := rows.Scan(&t.ID, &t.Title, &dueAt, &completed, &completedAt,
			&createdAt, &recurrencePattern, &recurrenceTime, &tagsJSON, &t.Profile)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		t.Completed = completed
		t.CreatedAt = toLocal(createdAt)
		if dueAt.Valid {
			lt := toLocal(dueAt.Time)
			t.DueAt = &lt
		}
		if completedAt.Valid {
			lt := toLocal(completedAt.Time)
			t.CompletedAt = &lt
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

// toLocal reinterprets a UTC time as local time (Dolt stores DATETIME without timezone).
func toLocal(t time.Time) time.Time {
	if t.Location() == time.UTC {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	}
	return t
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

func quote(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

func quoteTime(t time.Time) string {
	return quote(t.Truncate(time.Second).Format(timeLayout))
}

func quoteTimePtr(t *time.Time) string {
	if t == nil {
		return "NULL"
	}
	return quoteTime(*t)
}

func quoteNullStr(s string) string {
	if s == "" {
		return "NULL"
	}
	return quote(s)
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
