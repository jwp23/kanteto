package tui

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

// --- Pure detection tests for newlyDueTasks ---

func TestNewlyDueTasks_Empty(t *testing.T) {
	got := newlyDueTasks(nil, time.Now(), map[string]bool{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestNewlyDueTasks_FutureOnly(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	tasks := []task.Task{
		{ID: "a", DueAt: &future},
		{ID: "b", DueAt: &future},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{})
	if len(got) != 0 {
		t.Errorf("expected empty for future tasks, got %v", got)
	}
}

func TestNewlyDueTasks_OnePastDue(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	tasks := []task.Task{
		{ID: "a", DueAt: &past},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{})
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("expected [a], got %v", got)
	}
}

func TestNewlyDueTasks_AlreadyAlerted(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	tasks := []task.Task{
		{ID: "a", DueAt: &past},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{"a": true})
	if len(got) != 0 {
		t.Errorf("expected empty for already-alerted, got %v", got)
	}
}

func TestNewlyDueTasks_UndatedIgnored(t *testing.T) {
	now := time.Now()
	tasks := []task.Task{
		{ID: "a", DueAt: nil},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{})
	if len(got) != 0 {
		t.Errorf("expected empty for undated, got %v", got)
	}
}

func TestNewlyDueTasks_MultiplePastDue(t *testing.T) {
	now := time.Now()
	past1 := now.Add(-3 * time.Hour)
	past2 := now.Add(-2 * time.Hour)
	past3 := now.Add(-1 * time.Hour)
	tasks := []task.Task{
		{ID: "a", DueAt: &past1},
		{ID: "b", DueAt: &past2},
		{ID: "c", DueAt: &past3},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{})
	if len(got) != 3 {
		t.Errorf("expected 3 past-due, got %v", got)
	}
}

func TestNewlyDueTasks_ExactlyNow(t *testing.T) {
	now := time.Now()
	tasks := []task.Task{
		{ID: "a", DueAt: &now},
	}
	got := newlyDueTasks(tasks, now, map[string]bool{})
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("expected [a] for exactly-now, got %v", got)
	}
}

func TestNewlyDueTasks_MixedStates(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	tasks := []task.Task{
		{ID: "past", DueAt: &past},
		{ID: "future", DueAt: &future},
		{ID: "alerted", DueAt: &past},
		{ID: "undated", DueAt: nil},
		{ID: "past2", DueAt: &past},
	}
	alerted := map[string]bool{"alerted": true}
	got := newlyDueTasks(tasks, now, alerted)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %v", got)
	}
	ids := map[string]bool{}
	for _, id := range got {
		ids[id] = true
	}
	if !ids["past"] || !ids["past2"] {
		t.Errorf("expected past and past2, got %v", got)
	}
}

// --- Model integration tests ---

func TestDueAlert_TickDetectsNewlyDue(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	m.alertPlayer = &fakePlayer{}

	// Add a past-due task
	past := time.Now().Add(-time.Hour)
	tk, err := m.svc.Add("overdue alert", &past)
	if err != nil {
		t.Fatal(err)
	}

	// Send tick
	got, cmd := m.Update(tickMsg(time.Now()))
	gm := got.(model)

	if !gm.alertedIDs[tk.ID] {
		t.Errorf("expected task %s in alertedIDs", tk.ID)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (alert + tick)")
	}
}

func TestDueAlert_NoDoubleAlert(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	fp := &fakePlayer{}
	m.alertPlayer = fp

	past := time.Now().Add(-time.Hour)
	if _, err := m.svc.Add("overdue alert", &past); err != nil {
		t.Fatal(err)
	}

	// First tick — should alert
	got, _ := m.Update(tickMsg(time.Now()))
	m = got.(model)
	firstCount := fp.callCount

	// Second tick — should NOT alert again
	got, cmd := m.Update(tickMsg(time.Now()))
	m = got.(model)

	if fp.callCount != firstCount {
		t.Errorf("expected no additional Play() call, got %d total", fp.callCount)
	}
	// cmd should just be the tick reschedule, not a batch
	if cmd == nil {
		t.Error("expected tick reschedule cmd")
	}
}

func TestDueAlert_CompletedTaskNoAlert(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	fp := &fakePlayer{}
	m.alertPlayer = fp

	past := time.Now().Add(-time.Hour)
	tk, err := m.svc.Add("complete me", &past)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.svc.Complete(tk.ID); err != nil {
		t.Fatal(err)
	}

	got, _ := m.Update(tickMsg(time.Now()))
	gm := got.(model)

	if gm.alertedIDs[tk.ID] {
		t.Error("completed task should not trigger alert")
	}
	if fp.callCount != 0 {
		t.Errorf("expected 0 Play() calls, got %d", fp.callCount)
	}
}

func TestDueAlert_DeletedTaskNoAlert(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	fp := &fakePlayer{}
	m.alertPlayer = fp

	past := time.Now().Add(-time.Hour)
	tk, err := m.svc.Add("delete me", &past)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.svc.Delete(tk.ID); err != nil {
		t.Fatal(err)
	}

	got, _ := m.Update(tickMsg(time.Now()))
	gm := got.(model)

	if gm.alertedIDs[tk.ID] {
		t.Error("deleted task should not trigger alert")
	}
	if fp.callCount != 0 {
		t.Errorf("expected 0 Play() calls, got %d", fp.callCount)
	}
}

func TestDueAlert_FutureTaskNoAlert(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	fp := &fakePlayer{}
	m.alertPlayer = fp

	future := time.Now().Add(time.Hour)
	if _, err := m.svc.Add("future task", &future); err != nil {
		t.Fatal(err)
	}

	got, _ := m.Update(tickMsg(time.Now()))
	gm := got.(model)

	if len(gm.alertedIDs) != 0 {
		t.Errorf("expected empty alertedIDs, got %v", gm.alertedIDs)
	}
	if fp.callCount != 0 {
		t.Errorf("expected 0 Play() calls, got %d", fp.callCount)
	}
}

func TestDueAlert_TickRefreshesData(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	fp := &fakePlayer{}
	m.alertPlayer = fp

	// Task added after model construction
	past := time.Now().Add(-time.Hour)
	tk, err := m.svc.Add("late add", &past)
	if err != nil {
		t.Fatal(err)
	}

	// Tick should pick it up via refreshData
	got, _ := m.Update(tickMsg(time.Now()))
	gm := got.(model)

	if !gm.alertedIDs[tk.ID] {
		t.Errorf("expected task %s in alertedIDs after tick refresh", tk.ID)
	}
}

// --- Sound command tests ---

func TestPlayAlert_ReturnsAlertPlayedMsg(t *testing.T) {
	fp := &fakePlayer{}
	cmd := playAlert(fp)
	msg := cmd()
	if _, ok := msg.(alertPlayedMsg); !ok {
		t.Errorf("expected alertPlayedMsg, got %T", msg)
	}
}

func TestAlertPlayedMsg_NoOp(t *testing.T) {
	m := testDayModel(t)
	m.alertedIDs = make(map[string]bool)
	m.alertPlayer = &fakePlayer{}

	got, cmd := m.Update(alertPlayedMsg{})
	gm := got.(model)
	if cmd != nil {
		t.Error("expected nil cmd for alertPlayedMsg")
	}
	_ = gm // just verify no panic
}

func TestFakePlayer_Called(t *testing.T) {
	fp := &fakePlayer{}
	cmd := playAlert(fp)

	msg := cmd()
	if _, ok := msg.(alertPlayedMsg); !ok {
		t.Errorf("expected alertPlayedMsg, got %T", msg)
	}
	if fp.callCount != 1 {
		t.Errorf("expected 1 Play() call, got %d", fp.callCount)
	}
}

// --- Test helpers ---

type fakePlayer struct {
	callCount int
}

func (f *fakePlayer) Play() {
	f.callCount++
}

