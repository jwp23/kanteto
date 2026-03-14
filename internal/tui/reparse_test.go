package tui

import (
	"testing"
	"time"
)

func TestReparse_Confirm(t *testing.T) {
	m := testDayModel(t)

	// Add undated task with extractable deadline
	if _, err := m.svc.Add("deploy service tomorrow", nil); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	// Press R to start reparse
	got := sendKey(m, "R").(model)
	if !got.reparseConfirm {
		t.Fatal("R should enter reparse confirm mode")
	}
	if got.reparseResult == "" {
		t.Fatal("reparseResult should show confirmation prompt")
	}

	// Confirm with y
	got = sendKey(got, "y").(model)
	if got.reparseConfirm {
		t.Error("reparseConfirm should be false after y")
	}

	// Task should now have a due date and stripped title
	tasks, _ := m.svc.ListAll()
	var found bool
	for _, tk := range tasks {
		if tk.Title == "deploy service" {
			found = true
			if tk.DueAt == nil {
				t.Error("expected DueAt to be set after reparse")
			}
		}
	}
	if !found {
		t.Error("expected task with stripped title 'deploy service'")
	}
}

func TestReparse_Cancel(t *testing.T) {
	m := testDayModel(t)

	if _, err := m.svc.Add("deploy service tomorrow", nil); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "R").(model)
	if !got.reparseConfirm {
		t.Fatal("R should enter reparse confirm mode")
	}

	// Cancel with esc
	got = sendKey(got, "n").(model)
	if got.reparseConfirm {
		t.Error("reparseConfirm should be false after n")
	}

	// Task should be unchanged
	tasks, _ := m.svc.ListAll()
	for _, tk := range tasks {
		if tk.Title == "deploy service tomorrow" && tk.DueAt != nil {
			t.Error("task should not have been reparsed after cancel")
		}
	}
}

func TestReparse_NoUndated(t *testing.T) {
	m := testDayModel(t)

	// Add only dated tasks
	now := time.Now()
	due := now.Add(time.Hour)
	if _, err := m.svc.Add("dated task", &due); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "R").(model)
	if got.reparseConfirm {
		t.Error("should not enter confirm mode with no undated tasks")
	}
	if got.reparseResult == "" {
		t.Error("should show status message about no undated tasks")
	}
}
