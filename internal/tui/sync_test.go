package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeSyncer struct {
	pushErr   error
	pullErr   error
	hasRemote bool
}

func (f *fakeSyncer) Push() error          { return f.pushErr }
func (f *fakeSyncer) Pull() error          { return f.pullErr }
func (f *fakeSyncer) HasRemote(string) bool { return f.hasRemote }

func testDayModelWithSyncer(t *testing.T, syncer Syncer) model {
	t.Helper()
	svc := testService(t)
	now := time.Now()
	viewDate := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	m := model{
		svc:      svc,
		viewMode: dayView,
		viewDate: viewDate,
		width:    80,
		height:   24,
		syncer:   syncer,
	}
	m.refreshData()
	return m
}

func TestSyncPush_Keybinding(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)

	got, cmd := m.Update(teaKey("P"))
	gm := got.(model)
	if gm.syncStatus != "Pushing..." {
		t.Errorf("expected syncStatus 'Pushing...', got %q", gm.syncStatus)
	}
	if cmd == nil {
		t.Error("expected a non-nil Cmd for async push")
	}
}

func TestSyncPush_Result(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)

	got, _ := m.Update(syncResultMsg{op: "push"})
	gm := got.(model)
	if gm.syncStatus != "Push complete" {
		t.Errorf("expected 'Push complete', got %q", gm.syncStatus)
	}
}

func TestSyncPush_Error(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)

	got, _ := m.Update(syncResultMsg{op: "push", err: errors.New("push failed")})
	gm := got.(model)
	if gm.err == nil || gm.err.Error() != "push failed" {
		t.Errorf("expected error 'push failed', got %v", gm.err)
	}
}

func TestSyncPull_Keybinding(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)

	got, cmd := m.Update(teaKey("p"))
	gm := got.(model)
	if gm.syncStatus != "Pulling..." {
		t.Errorf("expected syncStatus 'Pulling...', got %q", gm.syncStatus)
	}
	if cmd == nil {
		t.Error("expected a non-nil Cmd for async pull")
	}
}

func TestSyncPull_RefreshesData(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)

	// Add a task, then simulate pull result
	now := time.Now()
	due := now.Add(5 * time.Minute)
	m.svc.Add("pre-pull task", &due)

	got, _ := m.Update(syncResultMsg{op: "pull"})
	gm := got.(model)
	if gm.syncStatus != "Pull complete" {
		t.Errorf("expected 'Pull complete', got %q", gm.syncStatus)
	}
	// After pull, data should be refreshed (task should appear in allTasks)
	if len(gm.allTasks) != 1 {
		t.Errorf("expected 1 task after pull refresh, got %d", len(gm.allTasks))
	}
}

func TestSync_NoSyncer(t *testing.T) {
	m := testDayModelWithSyncer(t, nil)

	got := sendKey(m, "P").(model)
	if got.err == nil {
		t.Error("expected error when syncer is nil")
	}

	got = sendKey(m, "p").(model)
	if got.err == nil {
		t.Error("expected error when syncer is nil")
	}
}

func TestSync_NoRemote(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: false}
	m := testDayModelWithSyncer(t, syncer)

	got := sendKey(m, "P").(model)
	if got.err == nil {
		t.Error("expected error when no remote configured")
	}

	got = sendKey(m, "p").(model)
	if got.err == nil {
		t.Error("expected error when no remote configured")
	}
}

func TestSyncClearStatus(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.syncStatus = "Push complete"

	got, _ := m.Update(clearSyncMsg{})
	gm := got.(model)
	if gm.syncStatus != "" {
		t.Errorf("expected empty syncStatus after clearSyncMsg, got %q", gm.syncStatus)
	}
}

// teaKey creates a tea.KeyMsg for a rune key press.
func teaKey(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}
