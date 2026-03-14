package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTagInput_Submit(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("tag me", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	// Enter tag mode
	got := sendKey(m, "t").(model)
	if !got.tagging {
		t.Fatal("t should enter tag mode")
	}

	// Type "work"
	for _, c := range "work" {
		got = sendKey(got, string(c)).(model)
	}
	if got.tagInput != "work" {
		t.Errorf("expected tagInput 'work', got %q", got.tagInput)
	}

	// Submit
	got = sendSpecialKey(got, tea.KeyEnter).(model)
	if got.tagging {
		t.Error("tagging should be false after enter")
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "work" {
		t.Errorf("expected Tags=[work], got %v", updated.Tags)
	}
}

func TestTagInput_Escape(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("no tag", &due)
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "t").(model)
	got = sendKey(got, "w").(model)
	got = sendKey(got, "o").(model)

	got = sendSpecialKey(got, tea.KeyEscape).(model)
	if got.tagging {
		t.Error("tagging should be false after escape")
	}
	if got.tagInput != "" {
		t.Errorf("tagInput should be empty, got %q", got.tagInput)
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 0 {
		t.Errorf("no tags should be added after escape, got %v", updated.Tags)
	}
}

func TestTagInput_NoTasks(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "t").(model)
	if got.tagging {
		t.Error("t should not enter tag mode with no tasks")
	}
}

func TestUntagInput_Submit(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("untag me", &due, "work", "urgent")
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	// Enter untag mode
	got := sendKey(m, "T").(model)
	if !got.untagging {
		t.Fatal("T should enter untag mode")
	}

	// Type "work"
	for _, c := range "work" {
		got = sendKey(got, string(c)).(model)
	}

	got = sendSpecialKey(got, tea.KeyEnter).(model)
	if got.untagging {
		t.Error("untagging should be false after enter")
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "urgent" {
		t.Errorf("expected Tags=[urgent], got %v", updated.Tags)
	}
}

func TestUntagInput_Escape(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	due := now.Add(5 * time.Minute)
	tk, err := m.svc.Add("keep tags", &due, "work")
	if err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	got := sendKey(m, "T").(model)
	got = sendKey(got, "w").(model)

	got = sendSpecialKey(got, tea.KeyEscape).(model)
	if got.untagging {
		t.Error("untagging should be false after escape")
	}

	updated, err := m.svc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "work" {
		t.Errorf("tags should be unchanged, got %v", updated.Tags)
	}
}

func TestUntagInput_NoTasks(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "T").(model)
	if got.untagging {
		t.Error("T should not enter untag mode with no tasks")
	}
}

