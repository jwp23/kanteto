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
		svc:        svc,
		viewMode:   dayView,
		viewDate:   viewDate,
		width:      80,
		height:     24,
		syncer:     syncer,
		alertedIDs: make(map[string]bool),
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

// --- Auto-sync tests ---

func TestAutoSync_MutationIncrementsGen(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	now := time.Now()
	due := now.Add(5 * time.Minute)
	m.svc.Add("test task", &due)
	m.refreshData()

	got, cmd := m.Update(teaKey(" "))
	gm := got.(model)
	if gm.autoSyncGen != 1 {
		t.Errorf("expected autoSyncGen=1, got %d", gm.autoSyncGen)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for auto-sync tick")
	}
}

func TestAutoSync_StaleTickIgnored(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 5

	got, cmd := m.Update(autoSyncTickMsg{gen: 3})
	gm := got.(model)
	if gm.autoSyncBusy {
		t.Error("stale tick should not start push")
	}
	if cmd != nil {
		t.Error("expected nil cmd for stale tick")
	}
}

func TestAutoSync_TickTriggersPush(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1

	got, cmd := m.Update(autoSyncTickMsg{gen: 1})
	gm := got.(model)
	if !gm.autoSyncBusy {
		t.Error("matching tick should start push")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for push")
	}
}

func TestAutoSync_DirtyRePush(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncBusy = true
	m.autoSyncDirty = true

	got, cmd := m.Update(autoSyncResultMsg{})
	gm := got.(model)
	if !gm.autoSyncBusy {
		t.Error("expected re-push when dirty")
	}
	if gm.autoSyncDirty {
		t.Error("dirty should be cleared before re-push")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for re-push")
	}
}

func TestAutoSync_QuitFlushes(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1
	m.autoSyncPushed = false

	got, cmd := m.Update(teaKey("q"))
	gm := got.(model)
	if !gm.quitting {
		t.Error("expected quitting=true for flush")
	}
	if gm.syncStatus != "Syncing..." {
		t.Errorf("expected syncStatus 'Syncing...', got %q", gm.syncStatus)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for flush push")
	}
}

func TestAutoSync_QuitWaitsForInFlight(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1
	m.autoSyncBusy = true

	got, cmd := m.Update(teaKey("q"))
	gm := got.(model)
	if !gm.quitting {
		t.Error("expected quitting=true")
	}
	if cmd == nil {
		t.Error("expected timeout cmd")
	}
	if !gm.autoSyncBusy {
		t.Error("should not start new push since one is in flight")
	}
}

func TestAutoSync_ForceQuit(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.quitting = true

	_, cmd := m.Update(teaKey("q"))
	if cmd == nil {
		t.Error("expected tea.Quit cmd on force quit")
	}
}

func TestAutoSync_QuitSkipsFlushWhenPushed(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1
	m.autoSyncPushed = true

	got, cmd := m.Update(teaKey("q"))
	gm := got.(model)
	if gm.quitting {
		t.Error("should not set quitting when already pushed")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestAutoSync_ManualPCancelsAutoSync(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 5

	got, _ := m.Update(teaKey("P"))
	gm := got.(model)
	if gm.autoSyncGen != 6 {
		t.Errorf("expected autoSyncGen=6 after P, got %d", gm.autoSyncGen)
	}
}

func TestAutoSync_NoRemoteNoPush(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: false}
	m := testDayModelWithSyncer(t, syncer)
	now := time.Now()
	due := now.Add(5 * time.Minute)
	m.svc.Add("test task", &due)
	m.refreshData()

	_, cmd := m.Update(teaKey(" "))
	if cmd != nil {
		t.Error("expected nil cmd when no remote configured")
	}
}

func TestAutoSync_ErrorShowsStatus(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncBusy = true

	got, cmd := m.Update(autoSyncResultMsg{err: errors.New("network error")})
	gm := got.(model)
	if gm.syncStatus != "Auto-sync failed" {
		t.Errorf("expected 'Auto-sync failed', got %q", gm.syncStatus)
	}
	if gm.err != nil {
		t.Error("auto-sync errors should not set m.err")
	}
	if cmd == nil {
		t.Error("expected clear status cmd")
	}
}

func TestAutoSync_PushedTickIgnored(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1
	m.autoSyncPushed = true

	got, cmd := m.Update(autoSyncTickMsg{gen: 1})
	gm := got.(model)
	if gm.autoSyncBusy {
		t.Error("should not start push when already pushed")
	}
	if cmd != nil {
		t.Error("expected nil cmd when already pushed")
	}
}

func TestAutoSync_TickSetsDirtyWhenBusy(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncGen = 1
	m.autoSyncBusy = true

	got, cmd := m.Update(autoSyncTickMsg{gen: 1})
	gm := got.(model)
	if !gm.autoSyncDirty {
		t.Error("tick should set dirty when push is in flight")
	}
	if cmd != nil {
		t.Error("expected nil cmd when busy")
	}
}

func TestAutoSync_ResultQuitsWhenQuitting(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncBusy = true
	m.quitting = true

	got, cmd := m.Update(autoSyncResultMsg{})
	gm := got.(model)
	if gm.autoSyncBusy {
		t.Error("autoSyncBusy should be cleared")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestAutoSync_MutationSetsDirtyWhenBusy(t *testing.T) {
	syncer := &fakeSyncer{hasRemote: true}
	m := testDayModelWithSyncer(t, syncer)
	m.autoSyncBusy = true

	now := time.Now()
	due := now.Add(5 * time.Minute)
	m.svc.Add("task1", &due)
	m.refreshData()

	got, _ := m.Update(teaKey(" "))
	gm := got.(model)
	if !gm.autoSyncDirty {
		t.Error("mutation while push in flight should set dirty")
	}
}
