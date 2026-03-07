package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func testService(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return task.NewService(s)
}

func sendKey(m tea.Model, key string) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

func sendSpecialKey(m tea.Model, keyType tea.KeyType) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated
}

func testModel(t *testing.T) model {
	t.Helper()
	// March 2026: starts on Sunday, 31 days
	viewDate := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.Local)
	return model{
		svc:            testService(t),
		viewMode:       monthView,
		viewDate:       viewDate,
		monthCursorDay: 15,
		monthTasks:     make(map[int][]task.Task),
		width:          80,
		height:         24,
	}
}

func TestMonthCursor_InitializesToCurrentDay(t *testing.T) {
	svc := testService(t)
	m := model{
		svc:      svc,
		viewMode: dayView,
		viewDate: time.Date(2026, time.March, 15, 0, 0, 0, 0, time.Local),
	}

	// Switch to month view
	updated := sendKey(m, "m")
	got := updated.(model)
	if got.monthCursorDay != 15 {
		t.Errorf("switching to month view should set cursor to day %d, got %d", 15, got.monthCursorDay)
	}
}

func TestMonthCursor_DownMovesOneWeek(t *testing.T) {
	m := testModel(t)
	updated := sendKey(m, "j")
	got := updated.(model)
	if got.monthCursorDay != 22 {
		t.Errorf("j should move cursor from 15 to 22, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_DownClampsToLastDay(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 29 // 29+7=36 > 31

	updated := sendKey(m, "j")
	got := updated.(model)
	if got.monthCursorDay != 31 {
		t.Errorf("j from day 29 should clamp to 31, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_UpMovesOneWeek(t *testing.T) {
	m := testModel(t)
	updated := sendKey(m, "k")
	got := updated.(model)
	if got.monthCursorDay != 8 {
		t.Errorf("k should move cursor from 15 to 8, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_UpClampsToFirstDay(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 5

	updated := sendKey(m, "k")
	got := updated.(model)
	if got.monthCursorDay != 1 {
		t.Errorf("k from day 5 should clamp to 1, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_RightMovesOneDay(t *testing.T) {
	m := testModel(t)
	updated := sendSpecialKey(m, tea.KeyRight)
	got := updated.(model)
	if got.monthCursorDay != 16 {
		t.Errorf("right should move cursor from 15 to 16, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_RightClampsToLastDay(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 31

	updated := sendSpecialKey(m, tea.KeyRight)
	got := updated.(model)
	if got.monthCursorDay != 31 {
		t.Errorf("right at last day should stay at 31, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_LeftMovesOneDay(t *testing.T) {
	m := testModel(t)
	updated := sendSpecialKey(m, tea.KeyLeft)
	got := updated.(model)
	if got.monthCursorDay != 14 {
		t.Errorf("left should move cursor from 15 to 14, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_LeftClampsToFirstDay(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 1

	updated := sendSpecialKey(m, tea.KeyLeft)
	got := updated.(model)
	if got.monthCursorDay != 1 {
		t.Errorf("left at day 1 should stay at 1, got %d", got.monthCursorDay)
	}
}

func TestMonthCursor_EnterDrillsIntoDayView(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 20

	updated := sendSpecialKey(m, tea.KeyEnter)
	got := updated.(model)
	if got.viewMode != dayView {
		t.Error("enter should switch to day view")
	}
	if got.viewDate.Day() != 20 {
		t.Errorf("enter should set viewDate to day 20, got day %d", got.viewDate.Day())
	}
	if got.viewDate.Month() != time.March {
		t.Errorf("enter should preserve month, got %s", got.viewDate.Month())
	}
}

func TestMonthCursor_HStillNavigatesMonth(t *testing.T) {
	m := testModel(t)
	updated := sendKey(m, "h")
	got := updated.(model)
	if got.viewDate.Month() != time.February {
		t.Errorf("h in month view should go to previous month, got %s", got.viewDate.Month())
	}
}

func TestMonthCursor_LStillNavigatesMonth(t *testing.T) {
	m := testModel(t)
	updated := sendKey(m, "l")
	got := updated.(model)
	if got.viewDate.Month() != time.April {
		t.Errorf("l in month view should go to next month, got %s", got.viewDate.Month())
	}
}

func TestMonthCursor_HighlightRendered(t *testing.T) {
	m := testModel(t)
	m.monthCursorDay = 15

	output := renderMonthView(m)
	if !strings.Contains(output, "[15]") {
		t.Errorf("month view should highlight day 15 with brackets, got:\n%s", output)
	}
}
