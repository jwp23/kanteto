# Kanteto Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI + TUI micro-task tracker with recurring schedules, audible reminders, and day/week/month views.

**Architecture:** Go binary with Cobra CLI, Bubble Tea TUI, and a background daemon sharing a SQLite database through a common service layer. NLP date parsing for natural schedule input.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Cobra, SQLite (modernc.org/sqlite), ULID, TOML config

**Design doc:** `docs/plans/2026-03-03-kanteto-design.md`

**Commit convention:** One-line messages, `feat:` / `test:` / `fix:` / `refactor:` prefixes

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/jwp23/kanteto`

**Step 2: Create main.go**

```go
package main

import "github.com/jwp23/kanteto/cmd"

func main() {
	cmd.Execute()
}
```

**Step 3: Create cmd/root.go with minimal Cobra root command**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kt",
	Short: "Kanteto — track small tasks and promises",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kanteto — run 'kt --help' for usage")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 4: Install dependencies and verify**

Run: `go get github.com/spf13/cobra && go build -o kt .`
Expected: Binary `kt` built, runs with help output.

**Step 5: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: scaffold Go project with Cobra root command"
```

---

### Task 2: XDG Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	// Unset XDG vars to test defaults
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")

	home, _ := os.UserHomeDir()

	p := DefaultPaths()
	if p.ConfigFile != filepath.Join(home, ".config", "kanteto", "config.toml") {
		t.Errorf("unexpected config path: %s", p.ConfigFile)
	}
	if p.DBFile != filepath.Join(home, ".local", "share", "kanteto", "kanteto.db") {
		t.Errorf("unexpected db path: %s", p.DBFile)
	}
	if p.PIDFile != filepath.Join(home, ".local", "share", "kanteto", "daemon.pid") {
		t.Errorf("unexpected pid path: %s", p.PIDFile)
	}
}

func TestXDGOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/testconfig")
	t.Setenv("XDG_DATA_HOME", "/tmp/testdata")

	p := DefaultPaths()
	if p.ConfigFile != "/tmp/testconfig/kanteto/config.toml" {
		t.Errorf("XDG_CONFIG_HOME not respected: %s", p.ConfigFile)
	}
	if p.DBFile != "/tmp/testdata/kanteto/kanteto.db" {
		t.Errorf("XDG_DATA_HOME not respected: %s", p.DBFile)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg := LoadConfig("")
	if cfg.Display.TimeFormat != "12h" {
		t.Errorf("expected 12h default, got %s", cfg.Display.TimeFormat)
	}
	if cfg.Display.WeekStart != "sunday" {
		t.Errorf("expected sunday default, got %s", cfg.Display.WeekStart)
	}
	if cfg.Urgency.GradientStart != "2h" {
		t.Errorf("expected 2h default, got %s", cfg.Urgency.GradientStart)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — package doesn't exist yet.

**Step 3: Write implementation**

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Paths struct {
	ConfigFile string
	DBFile     string
	PIDFile    string
}

type Config struct {
	Reminder ReminderConfig `toml:"reminder"`
	Display  DisplayConfig  `toml:"display"`
	Urgency  UrgencyConfig  `toml:"urgency"`
}

type ReminderConfig struct {
	Sound    string `toml:"sound"`
	LeadTime string `toml:"lead_time"`
}

type DisplayConfig struct {
	TimeFormat string `toml:"time_format"`
	WeekStart  string `toml:"week_start"`
}

type UrgencyConfig struct {
	GradientStart string `toml:"gradient_start"`
}

func DefaultPaths() Paths {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return Paths{
		ConfigFile: filepath.Join(configHome, "kanteto", "config.toml"),
		DBFile:     filepath.Join(dataHome, "kanteto", "kanteto.db"),
		PIDFile:    filepath.Join(dataHome, "kanteto", "daemon.pid"),
	}
}

func LoadConfig(path string) Config {
	cfg := Config{
		Reminder: ReminderConfig{LeadTime: "0m"},
		Display:  DisplayConfig{TimeFormat: "12h", WeekStart: "sunday"},
		Urgency:  UrgencyConfig{GradientStart: "2h"},
	}
	if path == "" {
		path = DefaultPaths().ConfigFile
	}
	toml.DecodeFile(path, &cfg)
	return cfg
}
```

**Step 4: Install dependency and run tests**

Run: `go get github.com/BurntSushi/toml && go test ./internal/config/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add XDG-compliant config package with TOML loading"
```

---

### Task 3: Task Domain Model

**Files:**
- Create: `internal/task/model.go`

**Step 1: Write the model**

```go
package task

import "time"

type Task struct {
	ID          string
	Title       string
	DueAt       *time.Time
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	RemindAt    *time.Time
	Reminded    bool
	Recurrence  *RecurrenceRule
}

type RecurrenceRule struct {
	Pattern string    // "weekdays", "weekly:fri", "daily", "monthly:15"
	Time    string    // "16:00"
	NextDue time.Time // precomputed next occurrence
}
```

No test needed — pure data struct with no logic.

**Step 2: Commit**

```bash
git add internal/task/model.go
git commit -m "feat: add Task and RecurrenceRule domain model"
```

---

### Task 4: SQLite Store — Schema & Migrations

**Files:**
- Create: `internal/store/sqlite.go`
- Create: `internal/store/sqlite_test.go`

**Step 1: Write the failing test**

```go
package store

import (
	"testing"
)

func TestNewStore_CreatesSchema(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify tables exist by querying sqlite_master
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected tasks table, got count=%d", count)
	}
}

func TestNewStore_WALMode(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	var mode string
	s.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if mode != "wal" && mode != "memory" {
		t.Errorf("expected WAL mode, got %s", mode)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL

**Step 3: Write implementation**

```go
package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		os.MkdirAll(filepath.Dir(dbPath), 0755)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS schema_version (version INTEGER);
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		due_at DATETIME,
		completed INTEGER NOT NULL DEFAULT 0,
		completed_at DATETIME,
		created_at DATETIME NOT NULL,
		remind_at DATETIME,
		reminded INTEGER NOT NULL DEFAULT 0,
		recurrence_pattern TEXT,
		recurrence_time TEXT,
		recurrence_next_due DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_due_at ON tasks(due_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_remind_at ON tasks(remind_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(completed);
	`
	_, err := db.Exec(schema)
	return err
}
```

**Step 4: Install dependency and run tests**

Run: `go get modernc.org/sqlite && go test ./internal/store/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add SQLite store with schema and WAL mode"
```

---

### Task 5: SQLite Store — CRUD Operations

**Files:**
- Modify: `internal/store/sqlite.go`
- Modify: `internal/store/sqlite_test.go`

**Step 1: Write failing tests for Add, Get, List, Update, Delete**

```go
func TestAddAndGet(t *testing.T) {
	s, _ := New(":memory:")
	defer s.Close()

	tk := &task.Task{
		ID:        "01HTEST000000000000000001",
		Title:     "Buy groceries",
		CreatedAt: time.Now(),
	}
	if err := s.Add(tk); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Title != "Buy groceries" {
		t.Errorf("expected 'Buy groceries', got '%s'", got.Title)
	}
}

func TestListByDateRange(t *testing.T) {
	s, _ := New(":memory:")
	defer s.Close()

	now := time.Now()
	today := &task.Task{ID: "01HTEST000000000000000001", Title: "Today task", DueAt: &now, CreatedAt: now}
	tomorrow := time.Now().Add(24 * time.Hour)
	tmrw := &task.Task{ID: "01HTEST000000000000000002", Title: "Tomorrow task", DueAt: &tomorrow, CreatedAt: now}

	s.Add(today)
	s.Add(tmrw)

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	tasks, err := s.ListByDateRange(start, end)
	if err != nil {
		t.Fatalf("ListByDateRange failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestDelete(t *testing.T) {
	s, _ := New(":memory:")
	defer s.Close()

	tk := &task.Task{ID: "01HTEST000000000000000001", Title: "Delete me", CreatedAt: time.Now()}
	s.Add(tk)
	s.Delete(tk.ID)

	_, err := s.Get(tk.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestListOverdue(t *testing.T) {
	s, _ := New(":memory:")
	defer s.Close()

	past := time.Now().Add(-2 * time.Hour)
	tk := &task.Task{ID: "01HTEST000000000000000001", Title: "Overdue", DueAt: &past, CreatedAt: time.Now()}
	s.Add(tk)

	tasks, err := s.ListOverdue()
	if err != nil {
		t.Fatalf("ListOverdue failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 overdue task, got %d", len(tasks))
	}
}

func TestMarkDone(t *testing.T) {
	s, _ := New(":memory:")
	defer s.Close()

	tk := &task.Task{ID: "01HTEST000000000000000001", Title: "Do this", CreatedAt: time.Now()}
	s.Add(tk)
	s.MarkDone(tk.ID)

	got, _ := s.Get(tk.ID)
	if !got.Completed {
		t.Error("expected task to be completed")
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL — methods don't exist yet.

**Step 3: Add CRUD methods to sqlite.go**

Implement: `Add`, `Get`, `Delete`, `MarkDone`, `ListByDateRange`, `ListOverdue`, `ListDueReminders`, `Update`. Each method maps Task struct fields to/from SQL rows, handling nullable fields with `sql.NullTime` / `sql.NullString`. Add a helper `scanTask` to DRY the row scanning.

**Step 4: Run tests**

Run: `go test ./internal/store/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: add CRUD operations to SQLite store"
```

---

### Task 6: ULID Generation

**Files:**
- Create: `internal/task/id.go`
- Create: `internal/task/id_test.go`

**Step 1: Write failing test**

```go
package task

import "testing"

func TestNewID_IsUnique(t *testing.T) {
	a := NewID()
	b := NewID()
	if a == b {
		t.Errorf("expected unique IDs, got %s twice", a)
	}
}

func TestNewID_Length(t *testing.T) {
	id := NewID()
	if len(id) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars: %s", len(id), id)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/task/ -v -run TestNewID`
Expected: FAIL

**Step 3: Implement**

```go
package task

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func NewID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
```

**Step 4: Install dependency and run tests**

Run: `go get github.com/oklog/ulid/v2 && go test ./internal/task/ -v -run TestNewID`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/task/id.go internal/task/id_test.go go.mod go.sum
git commit -m "feat: add ULID generation for task IDs"
```

---

### Task 7: Task Service Layer

**Files:**
- Create: `internal/task/service.go`
- Create: `internal/task/service_test.go`

**Step 1: Write failing tests**

Test the service methods: `Add`, `Complete`, `Snooze`, `ListForDay`, `ListForWeek`, `ListForMonth`. The service wraps the store and adds business logic (ULID generation, recurring task advancement, snooze duration parsing).

Key test cases:
- `TestServiceAdd` — creates task with generated ID, stores it
- `TestServiceComplete_OneOff` — marks done, sets CompletedAt
- `TestServiceComplete_Recurring` — advances to next occurrence instead of marking done
- `TestServiceSnooze` — pushes RemindAt forward by duration
- `TestServiceListForDay` — returns tasks for a given date plus overdue

**Step 2: Run test to verify it fails**

Run: `go test ./internal/task/ -v`
Expected: FAIL

**Step 3: Implement service.go**

The service takes a `Store` interface (for testability):

```go
package task

import "time"

type Store interface {
	Add(t *Task) error
	Get(id string) (*Task, error)
	Delete(id string) error
	MarkDone(id string) error
	Update(t *Task) error
	ListByDateRange(start, end time.Time) ([]Task, error)
	ListOverdue() ([]Task, error)
	ListNoDueDate() ([]Task, error)
	ListDueReminders() ([]Task, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}
```

Implement each method. For `Complete` on recurring tasks: parse the recurrence pattern, compute the next due date, update the task's `DueAt`, `RemindAt`, `Reminded`, and `Recurrence.NextDue`, then call `store.Update`.

**Step 4: Run tests**

Run: `go test ./internal/task/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/task/service.go internal/task/service_test.go
git commit -m "feat: add task service layer with business logic"
```

---

### Task 8: Recurrence Logic

**Files:**
- Create: `internal/task/recurrence.go`
- Create: `internal/task/recurrence_test.go`

**Step 1: Write failing tests**

```go
package task

import (
	"testing"
	"time"
)

func TestNextOccurrence_Daily(t *testing.T) {
	rule := &RecurrenceRule{Pattern: "daily", Time: "09:00"}
	from := time.Date(2026, 3, 3, 9, 0, 0, 0, time.Local)
	next := NextOccurrence(rule, from)
	expected := time.Date(2026, 3, 4, 9, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrence_Weekdays(t *testing.T) {
	rule := &RecurrenceRule{Pattern: "weekdays", Time: "16:00"}
	// Friday -> should skip to Monday
	fri := time.Date(2026, 3, 6, 16, 0, 0, 0, time.Local)
	next := NextOccurrence(rule, fri)
	expected := time.Date(2026, 3, 9, 16, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected Monday %v, got %v", expected, next)
	}
}

func TestNextOccurrence_WeeklyFri(t *testing.T) {
	rule := &RecurrenceRule{Pattern: "weekly:fri", Time: "17:00"}
	fri := time.Date(2026, 3, 6, 17, 0, 0, 0, time.Local)
	next := NextOccurrence(rule, fri)
	expected := time.Date(2026, 3, 13, 17, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected next Friday %v, got %v", expected, next)
	}
}

func TestNextOccurrence_Monthly(t *testing.T) {
	rule := &RecurrenceRule{Pattern: "monthly:15", Time: "10:00"}
	from := time.Date(2026, 3, 15, 10, 0, 0, 0, time.Local)
	next := NextOccurrence(rule, from)
	expected := time.Date(2026, 4, 15, 10, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/task/ -v -run TestNextOccurrence`
Expected: FAIL

**Step 3: Implement recurrence.go**

Parse the pattern string ("daily", "weekdays", "weekly:fri", "monthly:15") and compute the next occurrence from a given timestamp. Handle edge cases: Friday->Monday for weekdays, month-end wrapping for monthly.

**Step 4: Run tests**

Run: `go test ./internal/task/ -v -run TestNextOccurrence`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/task/recurrence.go internal/task/recurrence_test.go
git commit -m "feat: add recurrence pattern computation"
```

---

### Task 9: NLP Date Parsing

**Files:**
- Create: `internal/nlp/parse.go`
- Create: `internal/nlp/parse_test.go`

**Step 1: Write failing tests**

```go
package nlp

import (
	"testing"
	"time"
)

func TestParseDeadline(t *testing.T) {
	ref := time.Date(2026, 3, 3, 12, 0, 0, 0, time.Local)
	tests := []struct {
		input string
		month time.Month
		day   int
	}{
		{"march 11", time.March, 11},
		{"tomorrow", time.March, 4},
		{"next friday", time.March, 6},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDeadline(tt.input, ref)
			if err != nil {
				t.Fatalf("ParseDeadline(%q) error: %v", tt.input, err)
			}
			if result.Month() != tt.month || result.Day() != tt.day {
				t.Errorf("ParseDeadline(%q) = %v, want month=%v day=%d", tt.input, result, tt.month, tt.day)
			}
		})
	}
}

func TestParseRecurrence(t *testing.T) {
	tests := []struct {
		input   string
		pattern string
		time    string
	}{
		{"weekdays at 4pm", "weekdays", "16:00"},
		{"every friday at 5pm", "weekly:fri", "17:00"},
		{"daily at 9am", "daily", "09:00"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rule, err := ParseRecurrence(tt.input)
			if err != nil {
				t.Fatalf("ParseRecurrence(%q) error: %v", tt.input, err)
			}
			if rule.Pattern != tt.pattern {
				t.Errorf("pattern = %q, want %q", rule.Pattern, tt.pattern)
			}
			if rule.Time != tt.time {
				t.Errorf("time = %q, want %q", rule.Time, tt.time)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/nlp/ -v`
Expected: FAIL

**Step 3: Implement parse.go**

Two functions:
- `ParseDeadline(input string, ref time.Time) (time.Time, error)` — uses `olebedev/when` or `tj/go-naturaldate` for one-off dates
- `ParseRecurrence(input string) (*task.RecurrenceRule, error)` — custom parser for recurring patterns using regex matching for "weekdays at TIME", "every DAY at TIME", "daily at TIME", "monthly on DAY"

Time parsing: handle "4pm" -> "16:00", "9am" -> "09:00", "5:30pm" -> "17:30".

**Step 4: Install dependencies and run tests**

Run: `go get github.com/tj/go-naturaldate && go test ./internal/nlp/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/nlp/ go.mod go.sum
git commit -m "feat: add NLP date and recurrence parsing"
```

---

### Task 10: CLI — Add Command

**Files:**
- Create: `cmd/add.go`
- Modify: `cmd/root.go` (wire up service)

**Step 1: Implement cmd/add.go**

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"
)

var (
	addBy    string
	addEvery string
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := getService()
		title := strings.Join(args, " ")

		opts := task.AddOptions{Title: title}

		if addBy != "" {
			// parse one-off deadline
			deadline, err := nlp.ParseDeadline(addBy, time.Now())
			if err != nil {
				return fmt.Errorf("could not parse deadline %q: %w", addBy, err)
			}
			opts.DueAt = &deadline
		}

		if addEvery != "" {
			rule, err := nlp.ParseRecurrence(addEvery)
			if err != nil {
				return fmt.Errorf("could not parse schedule %q: %w", addEvery, err)
			}
			opts.Recurrence = rule
		}

		t, err := svc.Add(opts)
		if err != nil {
			return err
		}
		fmt.Printf("Added: %s (%s)\n", t.Title, t.ID[:8])
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addBy, "by", "", "deadline (e.g., \"march 11\", \"tomorrow\")")
	addCmd.Flags().StringVar(&addEvery, "every", "", "recurring schedule (e.g., \"weekdays at 4pm\")")
	rootCmd.AddCommand(addCmd)
}
```

**Step 2: Wire up service initialization in root.go**

Add a `getService()` helper that initializes config, store, and service lazily. Uses `sync.Once` for single initialization.

**Step 3: Build and manual test**

Run: `go build -o kt . && ./kt add "Buy groceries"`
Expected: Prints "Added: Buy groceries (01HXXXXX)"

Run: `./kt add "Weekly update" --every "weekdays at 4pm"`
Expected: Prints "Added: Weekly update (01HXXXXX)"

**Step 4: Commit**

```bash
git add cmd/add.go cmd/root.go
git commit -m "feat: add 'kt add' CLI command with deadline and recurrence"
```

---

### Task 11: CLI — Done, Snooze, Remove, Edit Commands

**Files:**
- Create: `cmd/done.go`
- Create: `cmd/snooze.go`
- Create: `cmd/rm.go`
- Create: `cmd/edit.go`

**Step 1: Implement each command**

Each follows the same Cobra pattern:
- `kt done <id>` — calls `svc.Complete(id)`, prints confirmation. For recurring tasks, prints next occurrence.
- `kt snooze <id> --for "1 hour"` — calls `svc.Snooze(id, duration)`, prints new reminder time.
- `kt rm <id>` — calls `svc.Delete(id)`, prints confirmation.
- `kt edit <id> --title "new title"` — calls `svc.Update(id, changes)`, prints updated task.

Accept both full ULID and prefix match (first 6-8 chars) for IDs.

**Step 2: Build and manual test**

Run: `./kt add "Test task" && ./kt done <id-prefix>`
Expected: Task marked complete.

**Step 3: Commit**

```bash
git add cmd/done.go cmd/snooze.go cmd/rm.go cmd/edit.go
git commit -m "feat: add done, snooze, rm, and edit CLI commands"
```

---

### Task 12: CLI — List / Day / Week / Month Views

**Files:**
- Create: `cmd/list.go`
- Create: `internal/task/format.go`

**Step 1: Implement formatted output**

`internal/task/format.go` — functions to render task lists as formatted terminal output with ANSI colors:
- `FormatDayView(tasks []Task, date time.Time, cfg config.Config) string`
- `FormatWeekView(tasks []Task, weekStart time.Time, cfg config.Config) string`
- `FormatMonthView(tasks []Task, month time.Month, year int, cfg config.Config) string`

Day view: groups into OVERDUE / TODAY / UPCOMING sections.
Week view: 7-column layout with day headers.
Month view: calendar grid with task counts per day.

**Step 2: Implement cmd/list.go**

Register four commands that share navigation flags:
- `kt list` / `kt day` — day view (default: today)
- `kt week` — week view (default: this week)
- `kt month` — month view (default: this month)
- Flags: `--next`/`-n`, `--prev`/`-p`, `--date`

**Step 3: Build and manual test**

Run: `./kt day` / `./kt week` / `./kt month`
Expected: Formatted output with color.

**Step 4: Commit**

```bash
git add cmd/list.go internal/task/format.go
git commit -m "feat: add day/week/month CLI views with navigation"
```

---

### Task 13: TUI — App Shell & Day View

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/day.go`
- Create: `internal/tui/styles.go`

**Step 1: Install Bubble Tea and Lip Gloss**

Run: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss`

**Step 2: Implement styles.go**

Define the urgency gradient function:
```go
func UrgencyColor(dueAt *time.Time) lipgloss.Color
```
Interpolate RGB from white -> yellow -> amber -> orange -> red based on time remaining. Uses the thresholds from the design: 2h, 1h, 30m, 15m, overdue.

**Step 3: Implement app.go — main Bubble Tea model**

```go
type Model struct {
	svc       *task.Service
	cfg       config.Config
	view      ViewMode       // Day, Week, Month
	cursor    int            // selected task index
	date      time.Time      // current view anchor date
	tasks     []task.Task    // loaded tasks
	overdue   []task.Task    // overdue tasks
	width     int
	height    int
}
```

Handles key events: `j/k` cursor, `h/l` time navigation, `d/w/m` view switching, `a` add, `space` done, `x` delete, `s` snooze, `q` quit, `?` help.

**Step 4: Implement day.go — day view rendering**

Renders the layout from the design doc: header bar, OVERDUE section (red), TODAY section, UPCOMING section, keybinding footer. Uses Lip Gloss for styling, urgency gradient for task colors.

**Step 5: Wire up TUI launch in cmd/root.go**

When `kt` is called with no subcommand, launch the Bubble Tea program instead of printing help.

**Step 6: Build and test**

Run: `go build -o kt . && ./kt`
Expected: TUI opens with day view showing any existing tasks.

**Step 7: Commit**

```bash
git add internal/tui/ cmd/root.go go.mod go.sum
git commit -m "feat: add TUI shell with day view and urgency gradient"
```

---

### Task 14: TUI — Week View

**Files:**
- Create: `internal/tui/week.go`

**Step 1: Implement week.go**

7-column grid layout. Each column is a day (Sun-Sat per config). Shows task titles truncated to fit column width. Current day column highlighted. Tasks colored by urgency. Pressing `Enter` on a day switches to day view for that date.

**Step 2: Build and test**

Run: `./kt` then press `w`
Expected: Week view renders with 7 columns.

**Step 3: Commit**

```bash
git add internal/tui/week.go
git commit -m "feat: add TUI week view"
```

---

### Task 15: TUI — Month View

**Files:**
- Create: `internal/tui/month.go`

**Step 1: Implement month.go**

Calendar grid (6 rows x 7 columns). Each cell shows day number and task count. Days with overdue tasks highlighted red. Current day highlighted. Pressing `Enter` on a day switches to day view. Week starts on Sunday per config.

**Step 2: Build and test**

Run: `./kt` then press `m`
Expected: Month calendar renders with task counts.

**Step 3: Commit**

```bash
git add internal/tui/month.go
git commit -m "feat: add TUI month view"
```

---

### Task 16: TUI — Inline Add & Snooze Prompts

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Add text input mode**

When user presses `a`, show an inline text input at the bottom of the TUI (using `charmbracelet/bubbles/textinput`). Parse the input with NLP: detect `--by` and `--every` patterns, or accept plain title for no-deadline tasks. When user presses `s`, show a snooze duration prompt.

**Step 2: Install bubbles dependency**

Run: `go get github.com/charmbracelet/bubbles`

**Step 3: Build and test**

Run: `./kt` then press `a`, type "Call Bob --by friday", press Enter.
Expected: Task appears in the list.

**Step 4: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add inline task add and snooze prompts in TUI"
```

---

### Task 17: Daemon — Background Reminder Process

**Files:**
- Create: `internal/daemon/daemon.go`
- Create: `internal/daemon/daemon_test.go`
- Create: `cmd/daemon.go`

**Step 1: Write failing test**

```go
package daemon

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

type mockStore struct {
	reminders []task.Task
	updated   []string
}

func (m *mockStore) ListDueReminders() ([]task.Task, error) {
	return m.reminders, nil
}

func (m *mockStore) MarkReminded(id string) error {
	m.updated = append(m.updated, id)
	return nil
}

func TestCheckReminders(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Minute)
	ms := &mockStore{
		reminders: []task.Task{
			{ID: "001", Title: "Test", RemindAt: &past},
		},
	}
	d := &Daemon{store: ms, playSound: func() {}}
	d.checkOnce()

	if len(ms.updated) != 1 || ms.updated[0] != "001" {
		t.Errorf("expected reminder marked, got %v", ms.updated)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/daemon/ -v`
Expected: FAIL

**Step 3: Implement daemon.go**

```go
package daemon

import (
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Daemon struct {
	store     ReminderStore
	interval  time.Duration
	playSound func()
	stop      chan struct{}
}

func New(store ReminderStore) *Daemon {
	return &Daemon{
		store:     store,
		interval:  30 * time.Second,
		playSound: defaultPlaySound,
		stop:      make(chan struct{}),
	}
}

func (d *Daemon) Run() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.checkOnce()
		case <-d.stop:
			return
		}
	}
}

func (d *Daemon) checkOnce() {
	tasks, _ := d.store.ListDueReminders()
	for _, t := range tasks {
		d.playSound()
		d.store.MarkReminded(t.ID)
	}
}

func defaultPlaySound() {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
	case "linux":
		exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/complete.oga").Run()
	}
}
```

**Step 4: Implement cmd/daemon.go**

Three subcommands: `kt daemon start` (forks, writes PID), `kt daemon stop` (reads PID, sends SIGTERM), `kt daemon status` (checks PID alive).

**Step 5: Run tests**

Run: `go test ./internal/daemon/ -v`
Expected: PASS

**Step 6: Build and test**

Run: `./kt daemon start && ./kt daemon status`
Expected: "Daemon running (PID: XXXXX)"

**Step 7: Commit**

```bash
git add internal/daemon/ cmd/daemon.go
git commit -m "feat: add background reminder daemon with sound playback"
```

---

### Task 18: Integration Testing

**Files:**
- Create: `internal/task/integration_test.go`

**Step 1: Write end-to-end test**

Test the full flow: create store -> create service -> add task with deadline -> add recurring task -> complete recurring task (verify it advances) -> list for day (verify results) -> snooze task (verify new time).

**Step 2: Run tests**

Run: `go test ./... -v`
Expected: All tests pass.

**Step 3: Commit**

```bash
git add internal/task/integration_test.go
git commit -m "test: add integration tests for full task lifecycle"
```

---

### Task 19: Polish — Help Text, Version, Error Messages

**Files:**
- Modify: `cmd/root.go`
- Modify: various `cmd/*.go`

**Step 1: Add version flag**

`kt --version` prints "kanteto v0.1.0"

**Step 2: Improve help text**

Add examples to each command's `Example` field in Cobra:
```
kt add "Call dentist" --by "march 11"
kt add "Weekly standup" --every "weekdays at 9am"
```

**Step 3: Add friendly error messages**

When ID prefix matches multiple tasks, list them. When NLP parsing fails, suggest the structured flag format.

**Step 4: Build and verify**

Run: `./kt --help` / `./kt add --help`
Expected: Clean, helpful output.

**Step 5: Commit**

```bash
git add cmd/
git commit -m "feat: add version flag, help examples, and friendly errors"
```

---

### Task 20: Final Build & Smoke Test

**Step 1: Run all tests**

Run: `go test ./... -v -count=1`
Expected: All pass.

**Step 2: Build release binary**

Run: `go build -o kt .`
Expected: Single binary, no CGO.

**Step 3: Smoke test the full workflow**

```bash
./kt add "Buy groceries"
./kt add "Weekly sync" --every "friday at 5pm"
./kt add "Dentist appointment" --by "march 11"
./kt list
./kt day
./kt week
./kt month
./kt done <id>
./kt daemon start
./kt daemon status
./kt daemon stop
./kt   # TUI opens
```

**Step 4: Commit any fixes**

```bash
git add -A
git commit -m "fix: address smoke test issues"
```
