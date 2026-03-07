package tui

import (
	"testing"
	"time"
)

func testDayModelWithTasks(t *testing.T) model {
	t.Helper()
	m := testDayModel(t)
	now := time.Now()

	// Add 3 tasks in the near future (all "today")
	for i := 1; i <= 3; i++ {
		due := now.Add(time.Duration(i) * time.Minute)
		if _, err := m.svc.Add("task"+string(rune('0'+i)), &due); err != nil {
			t.Fatal(err)
		}
	}
	m.refreshData()
	return m
}

func TestKeypress_JK(t *testing.T) {
	m := testDayModelWithTasks(t)
	if len(m.allTasks) < 3 {
		t.Fatalf("need at least 3 tasks, got %d", len(m.allTasks))
	}

	// j moves cursor down
	got := sendKey(m, "j").(model)
	if got.cursor != 1 {
		t.Errorf("j: expected cursor 1, got %d", got.cursor)
	}

	got = sendKey(got, "j").(model)
	if got.cursor != 2 {
		t.Errorf("j: expected cursor 2, got %d", got.cursor)
	}

	// j at end stays at end
	got = sendKey(got, "j").(model)
	if got.cursor != len(got.allTasks)-1 {
		t.Errorf("j at end: expected %d, got %d", len(got.allTasks)-1, got.cursor)
	}

	// k moves cursor up
	got = sendKey(got, "k").(model)
	if got.cursor != 1 {
		t.Errorf("k: expected cursor 1, got %d", got.cursor)
	}

	got = sendKey(got, "k").(model)
	if got.cursor != 0 {
		t.Errorf("k: expected cursor 0, got %d", got.cursor)
	}

	// k at start stays at 0
	got = sendKey(got, "k").(model)
	if got.cursor != 0 {
		t.Errorf("k at start: expected 0, got %d", got.cursor)
	}
}

func TestKeypress_SpaceCompletes(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()
	due := now.Add(5 * time.Minute)
	if _, err := m.svc.Add("complete me", &due); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	before := len(m.allTasks)
	got := sendKey(m, " ").(model)
	if len(got.allTasks) >= before {
		t.Error("space should complete the task, reducing allTasks count")
	}
}

func TestKeypress_XDeletes(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()
	due := now.Add(5 * time.Minute)
	if _, err := m.svc.Add("delete me", &due); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	before := len(m.allTasks)
	got := sendKey(m, "x").(model)
	if len(got.allTasks) >= before {
		t.Error("x should delete the task, reducing allTasks count")
	}
}

func TestKeypress_ViewSwitching(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "w").(model)
	if got.viewMode != weekView {
		t.Errorf("w: expected weekView, got %d", got.viewMode)
	}

	got = sendKey(got, "m").(model)
	if got.viewMode != monthView {
		t.Errorf("m: expected monthView, got %d", got.viewMode)
	}

	got = sendKey(got, "d").(model)
	if got.viewMode != dayView {
		t.Errorf("d: expected dayView, got %d", got.viewMode)
	}
}

func TestKeypress_TimeNav(t *testing.T) {
	m := testDayModel(t)
	origDay := m.viewDate

	// Day view: l advances 1 day
	got := sendKey(m, "l").(model)
	expected := origDay.AddDate(0, 0, 1)
	if !got.viewDate.Equal(expected) {
		t.Errorf("l in dayView: expected %s, got %s", expected.Format("Jan 2"), got.viewDate.Format("Jan 2"))
	}

	// h goes back 1 day
	got = sendKey(got, "h").(model)
	if !got.viewDate.Equal(origDay) {
		t.Errorf("h in dayView: expected %s, got %s", origDay.Format("Jan 2"), got.viewDate.Format("Jan 2"))
	}

	// Week view: l advances 7 days
	got = sendKey(m, "w").(model)
	got = sendKey(got, "l").(model)
	expected = origDay.AddDate(0, 0, 7)
	if !got.viewDate.Equal(expected) {
		t.Errorf("l in weekView: expected %s, got %s", expected.Format("Jan 2"), got.viewDate.Format("Jan 2"))
	}
}

func TestKeypress_TJumpsToday(t *testing.T) {
	m := testDayModel(t)

	// Navigate to the past
	m.viewDate = m.viewDate.AddDate(0, 0, -10)
	m.refreshData()

	got := sendKey(m, "t").(model)
	today := time.Now()
	if got.viewDate.Day() != today.Day() || got.viewDate.Month() != today.Month() {
		t.Errorf("t: expected today (%s), got %s", today.Format("Jan 2"), got.viewDate.Format("Jan 2"))
	}
}

func TestKeypress_HelpToggle(t *testing.T) {
	m := testDayModel(t)

	got := sendKey(m, "?").(model)
	if !got.showHelp {
		t.Error("? should set showHelp to true")
	}

	// Any key should dismiss help
	got = sendKey(got, "j").(model)
	if got.showHelp {
		t.Error("any key should dismiss help")
	}
}
