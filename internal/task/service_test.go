package task_test

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func newTestService(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return task.NewService(s)
}

func TestService_Add(t *testing.T) {
	svc := newTestService(t)

	tk, err := svc.Add("Buy groceries", nil)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if tk.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", tk.Title, "Buy groceries")
	}
	if tk.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestService_AddWithDueDate(t *testing.T) {
	svc := newTestService(t)

	due := time.Now().Add(2 * time.Hour)
	tk, err := svc.Add("Call dentist", &due)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if tk.DueAt == nil {
		t.Fatal("DueAt is nil, want non-nil")
	}
	if tk.RemindAt == nil {
		t.Fatal("RemindAt should be auto-set when DueAt is set")
	}
}

func TestService_Complete(t *testing.T) {
	svc := newTestService(t)

	tk, _ := svc.Add("Test task", nil)
	if err := svc.Complete(tk.ID); err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	tasks, _ := svc.ListAll()
	if len(tasks) != 0 {
		t.Errorf("expected 0 incomplete tasks, got %d", len(tasks))
	}
}

func TestService_Delete(t *testing.T) {
	svc := newTestService(t)

	tk, _ := svc.Add("To delete", nil)
	if err := svc.Delete(tk.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	tasks, _ := svc.ListAll()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestService_Snooze(t *testing.T) {
	svc := newTestService(t)

	due := time.Now().Add(time.Hour)
	tk, _ := svc.Add("Snooze me", &due)

	if err := svc.Snooze(tk.ID, 2*time.Hour); err != nil {
		t.Fatalf("Snooze() error: %v", err)
	}
}

func TestService_ListToday(t *testing.T) {
	svc := newTestService(t)

	now := time.Now()
	later := now.Add(2 * time.Hour)
	tomorrow := now.Add(48 * time.Hour)

	svc.Add("Today task", &later)
	svc.Add("Tomorrow task", &tomorrow)
	svc.Add("No date task", nil)

	tasks, err := svc.ListToday()
	if err != nil {
		t.Fatalf("ListToday() error: %v", err)
	}

	// Should only get the task due today
	if len(tasks) != 1 {
		t.Errorf("expected 1 today task, got %d", len(tasks))
	}
}
