package task_test

import (
	"testing"
	"time"
)

func TestIntegration_FullLifecycle(t *testing.T) {
	svc := newTestService(t)

	due := time.Now().Add(2 * time.Hour)
	tk, err := svc.Add("Full lifecycle task", &due)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// Verify it appears in ListAll.
	tasks, err := svc.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != tk.ID {
		t.Errorf("listed task ID = %q, want %q", tasks[0].ID, tk.ID)
	}

	// Complete it.
	if err := svc.Complete(tk.ID); err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	// ListAll should now be empty (only incomplete tasks).
	tasks, err = svc.ListAll()
	if err != nil {
		t.Fatalf("ListAll() after complete error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 incomplete tasks, got %d", len(tasks))
	}

	// Get should still return it as completed.
	got, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if !got.Completed {
		t.Error("expected task to be marked completed")
	}
}

func TestIntegration_RecurringAdvance(t *testing.T) {
	svc := newTestService(t)

	tk, err := svc.AddRecurring("Daily standup", "daily", "9am")
	if err != nil {
		t.Fatalf("AddRecurring() error: %v", err)
	}

	// Complete the recurring task (should advance, not mark complete).
	if err := svc.Complete(tk.ID); err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	// Task should still appear in ListAll because recurring tasks advance.
	tasks, err := svc.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after recurring advance, got %d", len(tasks))
	}

	// Verify the task is not marked completed and DueAt is in the future.
	got, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Completed {
		t.Error("recurring task should not be marked completed after advance")
	}
	if got.DueAt == nil {
		t.Fatal("DueAt is nil after recurring advance")
	}
	if !got.DueAt.After(time.Now()) {
		t.Errorf("DueAt should be in the future, got %v", *got.DueAt)
	}
}

func TestIntegration_Snooze(t *testing.T) {
	svc := newTestService(t)

	due := time.Now().Add(1 * time.Hour)
	tk, err := svc.Add("Snooze integration", &due)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	if err := svc.Snooze(tk.ID, 2*time.Hour); err != nil {
		t.Fatalf("Snooze() error: %v", err)
	}

	got, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.DueAt == nil {
		t.Fatal("DueAt is nil after snooze")
	}

	// New due should be ~3 hours from now (original 1h + snooze 2h).
	expected := time.Now().Add(3 * time.Hour)
	diff := got.DueAt.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 1*time.Second {
		t.Errorf("DueAt = %v, want ~%v (diff %v exceeds 1s tolerance)", *got.DueAt, expected, diff)
	}
}

func TestIntegration_DateRange(t *testing.T) {
	svc := newTestService(t)

	now := time.Now()
	todayDue := now.Add(1 * time.Hour)
	tomorrowDue := now.Add(25 * time.Hour)

	tk1, err := svc.Add("Due today+1h", &todayDue)
	if err != nil {
		t.Fatalf("Add(today) error: %v", err)
	}
	if _, err := svc.Add("Due tomorrow", &tomorrowDue); err != nil {
		t.Fatalf("Add(tomorrow) error: %v", err)
	}
	if _, err := svc.Add("Undated", nil); err != nil {
		t.Fatalf("Add(undated) error: %v", err)
	}

	// Query for today only.
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	tasks, err := svc.ListByDateRange(start, end)
	if err != nil {
		t.Fatalf("ListByDateRange() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in today's range, got %d", len(tasks))
	}
	if tasks[0].ID != tk1.ID {
		t.Errorf("returned task ID = %q, want %q", tasks[0].ID, tk1.ID)
	}
}

func TestIntegration_Overdue(t *testing.T) {
	svc := newTestService(t)

	pastDue := time.Now().Add(-1 * time.Hour)
	tk, err := svc.Add("Overdue task", &pastDue)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	tasks, err := svc.ListOverdue()
	if err != nil {
		t.Fatalf("ListOverdue() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 overdue task, got %d", len(tasks))
	}
	if tasks[0].ID != tk.ID {
		t.Errorf("overdue task ID = %q, want %q", tasks[0].ID, tk.ID)
	}

	// Complete it and verify overdue list is now empty.
	if err := svc.Complete(tk.ID); err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	tasks, err = svc.ListOverdue()
	if err != nil {
		t.Fatalf("ListOverdue() after complete error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 overdue tasks after complete, got %d", len(tasks))
	}
}

func TestIntegration_EditWorkflow(t *testing.T) {
	svc := newTestService(t)

	tk, err := svc.Add("Original title", nil)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// Get the task.
	got, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	// Change the title and update.
	got.Title = "Updated title"
	if err := svc.Update(got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// Get again and verify.
	got2, err := svc.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() after update error: %v", err)
	}
	if got2.Title != "Updated title" {
		t.Errorf("Title = %q, want %q", got2.Title, "Updated title")
	}
}
