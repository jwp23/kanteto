package store

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

const timeLayout = "2006-01-02 15:04:05"

const taskColumns = `id, title, due_at, completed, completed_at, created_at,
	recurrence_pattern, recurrence_time, tags, profile`

// Store is a Dolt-backed task repository that shells out to `dolt sql`.
type Store struct {
	dir string
}

// New opens (or creates) a Dolt database in dir and ensures the schema exists.
func New(dir string) (*Store, error) {
	if _, err := exec.LookPath("dolt"); err != nil {
		return nil, fmt.Errorf("dolt not found on PATH: install from https://docs.dolthub.com/introduction/installation")
	}

	s := &Store{dir: dir}

	if _, err := os.Stat(filepath.Join(dir, ".dolt")); os.IsNotExist(err) {
		if err := s.initRepo(); err != nil {
			return nil, fmt.Errorf("init dolt repo: %w", err)
		}
	}

	return s, nil
}

func (s *Store) initRepo() error {
	if err := s.runDolt("init"); err != nil {
		return err
	}

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
	if err := s.execSQL(schema); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	if err := s.runDolt("add", "-A"); err != nil {
		return err
	}
	return s.runDolt("commit", "-m", "init", "--allow-empty")
}

// Close is a no-op for the CLI store.
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
	return s.execSQL(q)
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
	return s.execSQL(q)
}

// Delete removes a task permanently.
func (s *Store) Delete(id string) error {
	q := fmt.Sprintf(`DELETE FROM tasks WHERE id = %s`, quote(id))
	return s.execSQL(q)
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
	return s.execSQL(q)
}

// UpdateDueAt changes a task's due date (for snooze).
func (s *Store) UpdateDueAt(id string, dueAt *time.Time) error {
	q := fmt.Sprintf(`UPDATE tasks SET due_at = %s WHERE id = %s`, quoteTimePtr(dueAt), quote(id))
	return s.execSQL(q)
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
	q := `SELECT DISTINCT profile FROM tasks ORDER BY profile`
	out, err := s.queryJSON(q)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rows []map[string]any `json:"rows"`
	}
	if out == "" || out == "{}" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	var profiles []string
	for _, row := range result.Rows {
		if p, ok := row["profile"].(string); ok {
			profiles = append(profiles, p)
		}
	}
	return profiles, nil
}

// runDolt executes a dolt CLI command (not sql).
func (s *Store) runDolt(args ...string) error {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dolt %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return nil
}

// execSQL runs a write query (INSERT/UPDATE/DELETE) — no JSON output needed.
func (s *Store) execSQL(q string) error {
	cmd := exec.Command("dolt", "sql", "-q", q)
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dolt sql: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// queryJSON runs a SELECT query with -r json and returns raw JSON stdout.
func (s *Store) queryJSON(q string) (string, error) {
	cmd := exec.Command("dolt", "sql", "-q", q, "-r", "json")
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dolt sql: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// queryTasks runs a SELECT query and parses the JSON result into tasks.
func (s *Store) queryTasks(q string) ([]task.Task, error) {
	out, err := s.queryJSON(q)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rows []map[string]any `json:"rows"`
	}
	if out == "" || out == "{}" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parse query result: %w", err)
	}

	tasks := make([]task.Task, 0, len(result.Rows))
	for _, row := range result.Rows {
		t, err := rowToTask(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func rowToTask(row map[string]any) (task.Task, error) {
	var t task.Task

	t.ID = strVal(row, "id")
	t.Title = strVal(row, "title")
	t.DueAt = timeVal(row, "due_at")
	t.Completed = intBool(row, "completed")
	t.CompletedAt = timeVal(row, "completed_at")

	if v := strVal(row, "created_at"); v != "" {
		parsed, err := time.ParseInLocation(timeLayout, v, time.Local)
		if err != nil {
			return t, fmt.Errorf("parse created_at: %w", err)
		}
		t.CreatedAt = parsed
	}

	t.RecurrencePattern = strVal(row, "recurrence_pattern")
	t.RecurrenceTime = strVal(row, "recurrence_time")
	t.Tags = tagsVal(row, "tags")
	t.Profile = strVal(row, "profile")

	return t, nil
}

func strVal(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func timeVal(row map[string]any, key string) *time.Time {
	s := strVal(row, key)
	if s == "" {
		return nil
	}
	parsed, err := time.ParseInLocation(timeLayout, s, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}

func intBool(row map[string]any, key string) bool {
	v, ok := row[key]
	if !ok || v == nil {
		return false
	}
	f, ok := v.(float64)
	if !ok {
		return false
	}
	return f != 0
}

func tagsVal(row map[string]any, key string) []string {
	v, ok := row[key]
	if !ok || v == nil {
		return []string{}
	}

	// Dolt returns JSON columns as native arrays
	arr, ok := v.([]any)
	if !ok {
		// Fallback: maybe it's a string
		s, ok := v.(string)
		if !ok {
			return []string{}
		}
		var tags []string
		if err := json.Unmarshal([]byte(s), &tags); err != nil {
			return []string{}
		}
		if tags == nil {
			return []string{}
		}
		return tags
	}

	tags := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			tags = append(tags, s)
		}
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
