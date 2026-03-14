package task_test

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func newTestService(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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

func TestService_AddWithTags(t *testing.T) {
	svc := newTestService(t)

	tk, err := svc.Add("Tagged task", nil, "work", "urgent")
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if len(tk.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tk.Tags))
	}
	if tk.Tags[0] != "work" || tk.Tags[1] != "urgent" {
		t.Errorf("Tags = %v, want [work urgent]", tk.Tags)
	}

	// Verify round-trip through store
	got, _ := svc.Get(tk.ID)
	if len(got.Tags) != 2 {
		t.Fatalf("after Get: expected 2 tags, got %d", len(got.Tags))
	}
}

func TestService_AddTag(t *testing.T) {
	svc := newTestService(t)

	tk, _ := svc.Add("Taggable", nil)
	if err := svc.AddTag(tk.ID, "work"); err != nil {
		t.Fatalf("AddTag() error: %v", err)
	}

	got, _ := svc.Get(tk.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "work" {
		t.Errorf("Tags = %v, want [work]", got.Tags)
	}

	// Adding duplicate tag should be a no-op
	if err := svc.AddTag(tk.ID, "work"); err != nil {
		t.Fatalf("AddTag() duplicate error: %v", err)
	}
	got, _ = svc.Get(tk.ID)
	if len(got.Tags) != 1 {
		t.Errorf("expected 1 tag after duplicate add, got %d", len(got.Tags))
	}
}

func TestService_RemoveTag(t *testing.T) {
	svc := newTestService(t)

	tk, _ := svc.Add("Untaggable", nil, "work", "urgent")
	if err := svc.RemoveTag(tk.ID, "work"); err != nil {
		t.Fatalf("RemoveTag() error: %v", err)
	}

	got, _ := svc.Get(tk.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("Tags = %v, want [urgent]", got.Tags)
	}

	// Removing non-existent tag should be a no-op
	if err := svc.RemoveTag(tk.ID, "nonexistent"); err != nil {
		t.Fatalf("RemoveTag() non-existent error: %v", err)
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
