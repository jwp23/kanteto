package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAddInput_Submit(t *testing.T) {
	m := testDayModel(t)

	// Enter add mode
	got := sendKey(m, "a").(model)
	if !got.adding {
		t.Fatal("a should enter add mode")
	}

	// Type "test task"
	for _, c := range "test task" {
		got = sendKey(got, string(c)).(model)
	}

	if got.addInput != "test task" {
		t.Errorf("expected addInput 'test task', got %q", got.addInput)
	}

	// Submit with enter
	got = sendSpecialKey(got, tea.KeyEnter).(model)

	if got.adding {
		t.Error("adding should be false after enter")
	}

	tasks, err := m.svc.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "test task" {
		t.Errorf("expected title 'test task', got %q", tasks[0].Title)
	}
}

func TestAddInput_Escape(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "a").(model)
	got = sendKey(got, "h").(model)
	got = sendKey(got, "i").(model)

	got = sendSpecialKey(got, tea.KeyEscape).(model)

	if got.adding {
		t.Error("adding should be false after escape")
	}
	if got.addInput != "" {
		t.Errorf("addInput should be empty after escape, got %q", got.addInput)
	}

	tasks, err := m.svc.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("no task should be created after escape, got %d", len(tasks))
	}
}

func TestSnoozeInput_Submit(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	// Add a task due soon
	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("snooze me", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	if len(m.allTasks) == 0 {
		t.Fatal("expected at least 1 task")
	}

	// Enter snooze mode
	got := sendKey(m, "s").(model)
	if !got.snoozing {
		t.Fatal("s should enter snooze mode")
	}

	// Type "1h"
	for _, c := range "1h" {
		got = sendKey(got, string(c)).(model)
	}

	got = sendSpecialKey(got, tea.KeyEnter).(model)

	if got.snoozing {
		t.Error("snoozing should be false after enter")
	}

	// Verify DueAt moved forward
	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DueAt == nil {
		t.Fatal("DueAt should not be nil")
	}
	// Should be roughly 1 hour later than original
	diff := updated.DueAt.Sub(due)
	if diff < 50*time.Minute || diff > 70*time.Minute {
		t.Errorf("expected ~1h snooze, got %v", diff)
	}
}

func TestEditInput_Submit(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("edit me", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	if len(m.allTasks) == 0 {
		t.Fatal("expected at least 1 task")
	}

	// Enter edit mode
	got := sendKey(m, "e").(model)
	if !got.editing {
		t.Fatal("e should enter edit mode")
	}

	// Type "tomorrow"
	for _, c := range "tomorrow" {
		got = sendKey(got, string(c)).(model)
	}

	got = sendSpecialKey(got, tea.KeyEnter).(model)

	if got.editing {
		t.Error("editing should be false after enter")
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DueAt == nil {
		t.Fatal("DueAt should not be nil")
	}
	if updated.DueAt.Day() == due.Day() && updated.DueAt.Month() == due.Month() {
		t.Error("DueAt should have changed to tomorrow")
	}
}

func TestEditInput_Escape(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("no edit", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "e").(model)
	got = sendKey(got, "f").(model)
	got = sendKey(got, "r").(model)

	got = sendSpecialKey(got, tea.KeyEscape).(model)

	if got.editing {
		t.Error("editing should be false after escape")
	}
	if got.editInput != "" {
		t.Errorf("editInput should be empty, got %q", got.editInput)
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.DueAt.Equal(due) {
		t.Errorf("DueAt should be unchanged, was %v now %v", due, updated.DueAt)
	}
}

func TestEditInput_NoTasks(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "e").(model)
	if got.editing {
		t.Error("e should not enter edit mode with no tasks")
	}
}

func TestSnoozeInput_Escape(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("no snooze", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "s").(model)
	got = sendKey(got, "2").(model)
	got = sendKey(got, "h").(model)

	got = sendSpecialKey(got, tea.KeyEscape).(model)

	if got.snoozing {
		t.Error("snoozing should be false after escape")
	}
	if got.snoozeInput != "" {
		t.Errorf("snoozeInput should be empty, got %q", got.snoozeInput)
	}

	// DueAt unchanged
	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.DueAt.Equal(due) {
		t.Errorf("DueAt should be unchanged, was %v now %v", due, updated.DueAt)
	}
}
