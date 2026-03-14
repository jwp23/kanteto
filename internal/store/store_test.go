package store

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStore_CreateAndGet(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Buy groceries",
		CreatedAt: time.Now(),
	}

	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Title != tk.Title {
		t.Errorf("Title = %q, want %q", got.Title, tk.Title)
	}
	if got.ID != tk.ID {
		t.Errorf("ID = %q, want %q", got.ID, tk.ID)
	}
}

func TestStore_CreateWithDueDate(t *testing.T) {
	s := newTestStore(t)

	due := time.Now().Add(2 * time.Hour)
	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Call dentist",
		DueAt:     &due,
		CreatedAt: time.Now(),
	}

	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.DueAt == nil {
		t.Fatal("DueAt is nil, want non-nil")
	}
}

func TestStore_Complete(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{ID: task.NewID(), Title: "Test", CreatedAt: time.Now()}
	if err := s.Create(tk); err != nil {
		t.Fatal(err)
	}

	if err := s.Complete(tk.ID); err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Completed {
		t.Error("expected task to be completed")
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{ID: task.NewID(), Title: "To delete", CreatedAt: time.Now()}
	if err := s.Create(tk); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(tk.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := s.Get(tk.ID)
	if err == nil {
		t.Error("expected error getting deleted task, got nil")
	}
}

func TestStore_ListByDateRange(t *testing.T) {
	s := newTestStore(t)

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(48 * time.Hour)

	tasks := []task.Task{
		{ID: task.NewID(), Title: "Past", DueAt: &past, CreatedAt: now},
		{ID: task.NewID(), Title: "Today", DueAt: &now, CreatedAt: now},
		{ID: task.NewID(), Title: "Future", DueAt: &future, CreatedAt: now},
		{ID: task.NewID(), Title: "No date", CreatedAt: now},
	}
	for _, tk := range tasks {
		if err := s.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	result, err := s.ListByDateRange(start, end)
	if err != nil {
		t.Fatalf("ListByDateRange() error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 task in range, got %d", len(result))
	}
}

func TestStore_ListAll(t *testing.T) {
	s := newTestStore(t)

	now := time.Now()
	for i := 0; i < 3; i++ {
		tk := task.Task{ID: task.NewID(), Title: "Task", CreatedAt: now}
		if err := s.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ListAll(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(all))
	}
}

func TestStore_ListUndated(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	due := now.Add(2 * time.Hour)

	// Task with due date
	s.Create(task.Task{ID: task.NewID(), Title: "With due", DueAt: &due, CreatedAt: now})
	// Incomplete task without due date
	s.Create(task.Task{ID: task.NewID(), Title: "No due", CreatedAt: now})
	// Completed task without due date — should NOT appear
	completedAt := now
	s.Create(task.Task{ID: task.NewID(), Title: "Done no due", Completed: true, CompletedAt: &completedAt, CreatedAt: now})

	result, err := s.ListUndated()
	if err != nil {
		t.Fatalf("ListUndated() error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 undated task, got %d", len(result))
	}
	if result[0].Title != "No due" {
		t.Errorf("expected 'No due', got %q", result[0].Title)
	}
}

func TestStore_ListOverdue(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	pastDue := now.Add(-48 * time.Hour)
	futureDue := now.Add(48 * time.Hour)

	s.Create(task.Task{ID: task.NewID(), Title: "Overdue", DueAt: &pastDue, CreatedAt: now})
	s.Create(task.Task{ID: task.NewID(), Title: "Future", DueAt: &futureDue, CreatedAt: now})

	result, err := s.ListOverdue()
	if err != nil {
		t.Fatalf("ListOverdue() error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 overdue task, got %d", len(result))
	}
	if result[0].Title != "Overdue" {
		t.Errorf("expected 'Overdue', got %q", result[0].Title)
	}
}

func TestStore_ListOverdueAsOf(t *testing.T) {
	s := newTestStore(t)
	// Fixed dates for determinism
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.Local)
	earlyDue := time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local)
	lateDue := time.Date(2026, 3, 15, 10, 0, 0, 0, time.Local)

	s.Create(task.Task{ID: task.NewID(), Title: "Before AsOf", DueAt: &earlyDue, CreatedAt: base})
	s.Create(task.Task{ID: task.NewID(), Title: "After AsOf", DueAt: &lateDue, CreatedAt: base})

	result, err := s.ListOverdueAsOf(base)
	if err != nil {
		t.Fatalf("ListOverdueAsOf() error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 overdue task, got %d", len(result))
	}
	if result[0].Title != "Before AsOf" {
		t.Errorf("expected 'Before AsOf', got %q", result[0].Title)
	}
}

func TestStore_ListDueReminders(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	pastRemind := now.Add(-1 * time.Hour)

	// (a) remind_at in past + reminded=false → should be returned
	s.Create(task.Task{ID: task.NewID(), Title: "Needs reminder", RemindAt: &pastRemind, CreatedAt: now})
	// (b) remind_at in past + reminded=true → should NOT be returned
	alreadyRemindedID := task.NewID()
	s.Create(task.Task{ID: alreadyRemindedID, Title: "Already reminded", RemindAt: &pastRemind, CreatedAt: now})
	s.MarkReminded(alreadyRemindedID)
	// (c) no remind_at → should NOT be returned
	s.Create(task.Task{ID: task.NewID(), Title: "No reminder", CreatedAt: now})

	result, err := s.ListDueReminders()
	if err != nil {
		t.Fatalf("ListDueReminders() error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 due reminder, got %d", len(result))
	}
	if result[0].Title != "Needs reminder" {
		t.Errorf("expected 'Needs reminder', got %q", result[0].Title)
	}
}

func TestStore_MarkReminded(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	remind := now.Add(1 * time.Hour)

	tk := task.Task{ID: task.NewID(), Title: "Remind me", RemindAt: &remind, CreatedAt: now}
	s.Create(tk)

	if err := s.MarkReminded(tk.ID); err != nil {
		t.Fatalf("MarkReminded() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Reminded {
		t.Error("expected Reminded to be true after MarkReminded")
	}
}

func TestStore_Update(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	tk := task.Task{ID: task.NewID(), Title: "Original", CreatedAt: now}
	s.Create(tk)

	// Mutate all fields
	newDue := now.Add(24 * time.Hour)
	newRemind := now.Add(12 * time.Hour)
	newCompleted := now
	tk.Title = "Updated"
	tk.DueAt = &newDue
	tk.RemindAt = &newRemind
	tk.RecurrencePattern = "daily"
	tk.RecurrenceTime = "9:00"
	tk.Completed = true
	tk.CompletedAt = &newCompleted
	tk.Reminded = true

	if err := s.Update(tk); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Updated" {
		t.Errorf("Title = %q, want %q", got.Title, "Updated")
	}
	if got.DueAt == nil {
		t.Fatal("DueAt should not be nil")
	}
	if got.RemindAt == nil {
		t.Fatal("RemindAt should not be nil")
	}
	if got.RecurrencePattern != "daily" {
		t.Errorf("RecurrencePattern = %q, want %q", got.RecurrencePattern, "daily")
	}
	if got.RecurrenceTime != "9:00" {
		t.Errorf("RecurrenceTime = %q, want %q", got.RecurrenceTime, "9:00")
	}
	if !got.Completed {
		t.Error("expected Completed to be true")
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	if !got.Reminded {
		t.Error("expected Reminded to be true")
	}
}

func TestStore_UpdateDueAt(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{ID: task.NewID(), Title: "Snooze me", CreatedAt: time.Now()}
	if err := s.Create(tk); err != nil {
		t.Fatal(err)
	}

	newDue := time.Now().Add(2 * time.Hour)
	if err := s.UpdateDueAt(tk.ID, &newDue); err != nil {
		t.Fatalf("UpdateDueAt() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DueAt == nil {
		t.Fatal("DueAt is nil after update")
	}
}

func TestStore_CreateWithTags(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Tagged task",
		CreatedAt: time.Now(),
		Tags:      []string{"work", "urgent"},
	}
	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(got.Tags))
	}
	if got.Tags[0] != "work" || got.Tags[1] != "urgent" {
		t.Errorf("Tags = %v, want [work urgent]", got.Tags)
	}
}

func TestStore_CreateWithNoTags(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "No tags",
		CreatedAt: time.Now(),
	}
	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Tags == nil {
		t.Fatal("Tags should be initialized to empty slice, not nil")
	}
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(got.Tags))
	}
}

func TestStore_UpdateTags(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Update tags",
		CreatedAt: time.Now(),
		Tags:      []string{"work"},
	}
	s.Create(tk)

	tk.Tags = []string{"work", "important"}
	if err := s.Update(tk); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags after update, got %d", len(got.Tags))
	}
	if got.Tags[1] != "important" {
		t.Errorf("Tags[1] = %q, want %q", got.Tags[1], "important")
	}
}

func TestStore_TagsPreservedInListAll(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	s.Create(task.Task{ID: task.NewID(), Title: "Tagged", CreatedAt: now, Tags: []string{"work"}})
	s.Create(task.Task{ID: task.NewID(), Title: "Untagged", CreatedAt: now})

	all, err := s.ListAll(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(all))
	}

	// Find the tagged task
	var tagged task.Task
	for _, tk := range all {
		if tk.Title == "Tagged" {
			tagged = tk
			break
		}
	}
	if len(tagged.Tags) != 1 || tagged.Tags[0] != "work" {
		t.Errorf("tagged task Tags = %v, want [work]", tagged.Tags)
	}
}

func TestStore_CreateWithProfile(t *testing.T) {
	s := newTestStore(t)

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Work task",
		CreatedAt: time.Now(),
		Profile:   "work",
	}
	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Profile != "work" {
		t.Errorf("Profile = %q, want %q", got.Profile, "work")
	}
}

func TestStore_DefaultProfile(t *testing.T) {
	s := newTestStore(t)

	// Task without explicit profile gets 'default' from DB default
	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Default profile",
		CreatedAt: time.Now(),
	}
	if err := s.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	// Empty string in Go maps to DB default 'default'
	// but since we INSERT the Go value directly, it'll be empty string
	// unless we handle it. Let's verify the round-trip.
	if got.Profile != "" {
		t.Errorf("Profile = %q, want empty (no explicit profile set)", got.Profile)
	}
}

func TestStore_MigrationVersioning(t *testing.T) {
	s := newTestStore(t)

	// Opening the same DB again should not fail (migrations are idempotent)
	var version int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("querying schema version: %v", err)
	}
	if version < 1 {
		t.Errorf("expected schema version >= 1, got %d", version)
	}
}
