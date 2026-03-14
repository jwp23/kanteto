package store

import (
	"os/exec"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not found on PATH, skipping integration test")
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	skipIfNoDolt(t)

	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
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

	due := time.Now().Add(2 * time.Hour).Truncate(time.Second)
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

	now := time.Now().Truncate(time.Second)
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
	now := time.Now().Truncate(time.Second)
	due := now.Add(2 * time.Hour)

	s.Create(task.Task{ID: task.NewID(), Title: "With due", DueAt: &due, CreatedAt: now})
	s.Create(task.Task{ID: task.NewID(), Title: "No due", CreatedAt: now})
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
	now := time.Now().Truncate(time.Second)
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
	now := time.Now().Truncate(time.Second)
	pastRemind := now.Add(-1 * time.Hour)

	s.Create(task.Task{ID: task.NewID(), Title: "Needs reminder", RemindAt: &pastRemind, CreatedAt: now})
	alreadyRemindedID := task.NewID()
	s.Create(task.Task{ID: alreadyRemindedID, Title: "Already reminded", RemindAt: &pastRemind, CreatedAt: now})
	s.MarkReminded(alreadyRemindedID)
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
	now := time.Now().Truncate(time.Second)
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
	now := time.Now().Truncate(time.Second)

	tk := task.Task{ID: task.NewID(), Title: "Original", CreatedAt: now}
	s.Create(tk)

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

	newDue := time.Now().Add(2 * time.Hour).Truncate(time.Second)
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
	if got.Profile != "" {
		t.Errorf("Profile = %q, want empty (no explicit profile set)", got.Profile)
	}
}

func TestStore_ListProfiles(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	s.Create(task.Task{ID: task.NewID(), Title: "Work", CreatedAt: now, Profile: "work"})
	s.Create(task.Task{ID: task.NewID(), Title: "Personal", CreatedAt: now, Profile: "personal"})
	s.Create(task.Task{ID: task.NewID(), Title: "Also work", CreatedAt: now, Profile: "work"})

	profiles, err := s.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d: %v", len(profiles), profiles)
	}
	if profiles[0] != "personal" || profiles[1] != "work" {
		t.Errorf("profiles = %v, want [personal work]", profiles)
	}
}

func TestStore_Close(t *testing.T) {
	s := newTestStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestStore_ListAllIncludesCompleted(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	s.Create(task.Task{ID: task.NewID(), Title: "Active", CreatedAt: now})
	completedID := task.NewID()
	s.Create(task.Task{ID: completedID, Title: "Done", CreatedAt: now})
	s.Complete(completedID)

	withCompleted, err := s.ListAll(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(withCompleted) != 2 {
		t.Errorf("ListAll(true): expected 2, got %d", len(withCompleted))
	}

	withoutCompleted, err := s.ListAll(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutCompleted) != 1 {
		t.Errorf("ListAll(false): expected 1, got %d", len(withoutCompleted))
	}
}
