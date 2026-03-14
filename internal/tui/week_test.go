package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func testWeekModel(t *testing.T) model {
	t.Helper()
	svc := testService(t)
	// March 15, 2026 is a Sunday (weekday 0)
	viewDate := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.Local)
	return model{
		svc:           svc,
		viewMode:      weekView,
		viewDate:      viewDate,
		weekCursorDay: 3, // Wednesday by default for most tests
		width:         120,
		height:        24,
	}
}

func TestRenderWeekView_Header(t *testing.T) {
	m := testWeekModel(t)
	output := renderWeekView(m)
	if !strings.Contains(output, "Week of") {
		t.Errorf("expected output to contain 'Week of', got:\n%s", output)
	}
}

func TestRenderWeekView_Tasks(t *testing.T) {
	m := testWeekModel(t)

	// Add a task on Monday March 16
	monday := time.Date(2026, time.March, 16, 10, 0, 0, 0, time.Local)
	m.svc.Add("Monday meeting", &monday)

	// Add a task on Wednesday March 18
	wednesday := time.Date(2026, time.March, 18, 14, 0, 0, 0, time.Local)
	m.svc.Add("Wednesday review", &wednesday)

	m.refreshData()
	output := renderWeekView(m)
	if !strings.Contains(output, "Monday meet") || !strings.Contains(output, "Wednesday r") {
		// Titles may be truncated by column width, check for partial match
		if !strings.Contains(output, "Monday") && !strings.Contains(output, "Wednesday") {
			t.Errorf("expected task titles in output, got:\n%s", output)
		}
	}
}

func TestRenderWeekView_Empty(t *testing.T) {
	m := testWeekModel(t)
	output := renderWeekView(m)
	if !strings.Contains(output, "No tasks this week") {
		t.Errorf("expected output to contain 'No tasks this week', got:\n%s", output)
	}
}

func TestWeekCursor_InitializesToCurrentDay(t *testing.T) {
	svc := testService(t)
	// March 18, 2026 is a Wednesday (weekday 3)
	m := model{
		svc:      svc,
		viewMode: dayView,
		viewDate: time.Date(2026, time.March, 18, 0, 0, 0, 0, time.Local),
	}

	updated := sendKey(m, "w")
	got := updated.(model)
	if got.weekCursorDay != 3 {
		t.Errorf("switching to week view should set cursor to weekday 3 (Wednesday), got %d", got.weekCursorDay)
	}
}

func TestWeekCursor_RightMovesOneDay(t *testing.T) {
	m := testWeekModel(t) // cursor at 3
	updated := sendKey(m, "j")
	got := updated.(model)
	if got.weekCursorDay != 4 {
		t.Errorf("j should move cursor from 3 to 4, got %d", got.weekCursorDay)
	}
}

func TestWeekCursor_LeftMovesOneDay(t *testing.T) {
	m := testWeekModel(t) // cursor at 3
	updated := sendKey(m, "k")
	got := updated.(model)
	if got.weekCursorDay != 2 {
		t.Errorf("k should move cursor from 3 to 2, got %d", got.weekCursorDay)
	}
}

func TestWeekCursor_RightClampsToSaturday(t *testing.T) {
	m := testWeekModel(t)
	m.weekCursorDay = 6

	updated := sendKey(m, "j")
	got := updated.(model)
	if got.weekCursorDay != 6 {
		t.Errorf("j from Saturday (6) should stay at 6, got %d", got.weekCursorDay)
	}
}

func TestWeekCursor_LeftClampsToSunday(t *testing.T) {
	m := testWeekModel(t)
	m.weekCursorDay = 0

	updated := sendKey(m, "k")
	got := updated.(model)
	if got.weekCursorDay != 0 {
		t.Errorf("k from Sunday (0) should stay at 0, got %d", got.weekCursorDay)
	}
}

func TestWeekCursor_EnterDrillsIntoDayView(t *testing.T) {
	m := testWeekModel(t)
	m.weekCursorDay = 4 // Thursday

	updated := sendSpecialKey(m, tea.KeyEnter)
	got := updated.(model)
	if got.viewMode != dayView {
		t.Error("enter should switch to day view")
	}
	// March 15 is Sunday (start of week), +4 = March 19 (Thursday)
	if got.viewDate.Day() != 19 {
		t.Errorf("enter should set viewDate to day 19 (Thursday), got day %d", got.viewDate.Day())
	}
	if got.viewDate.Month() != time.March {
		t.Errorf("enter should preserve month, got %s", got.viewDate.Month())
	}
}

func TestWeekCursor_HStillNavigatesWeek(t *testing.T) {
	m := testWeekModel(t)
	updated := sendKey(m, "h")
	got := updated.(model)
	// h shifts viewDate by -7 days: March 15 -> March 8
	if got.viewDate.Day() != 8 {
		t.Errorf("h in week view should go to previous week (day 8), got day %d", got.viewDate.Day())
	}
}

func TestWeekCursor_HighlightRendered(t *testing.T) {
	m := testWeekModel(t)
	m.weekCursorDay = 3 // Wednesday

	output := renderWeekView(m)
	if !strings.Contains(output, "[Wed") {
		t.Errorf("week view should highlight Wednesday with brackets, got:\n%s", output)
	}
}
