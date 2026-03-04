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
