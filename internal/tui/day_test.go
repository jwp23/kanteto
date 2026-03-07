package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

func testDayModel(t *testing.T) model {
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
	}
	m.refreshData()
	return m
}

func TestDayView_Sections(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	// Overdue: due yesterday — clearly in the past
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 10, 0, 0, 0, time.Local)
	if _, err := m.svc.Add("overdue task", &yesterday); err != nil {
		t.Fatal(err)
	}

	// Today: due 1 minute from now (guaranteed future within today)
	todaySoon := now.Add(1 * time.Minute)
	if _, err := m.svc.Add("today task", &todaySoon); err != nil {
		t.Fatal(err)
	}

	// Upcoming: due tomorrow
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 10, 0, 0, 0, time.Local)
	if _, err := m.svc.Add("upcoming task", &tomorrow); err != nil {
		t.Fatal(err)
	}

	// Undated
	if _, err := m.svc.Add("undated task", nil); err != nil {
		t.Fatal(err)
	}

	m.refreshData()

	if len(m.overdue) != 1 {
		t.Errorf("expected 1 overdue, got %d", len(m.overdue))
	}
	if len(m.today) != 1 {
		t.Errorf("expected 1 today, got %d", len(m.today))
	}
	if len(m.upcoming) != 1 {
		t.Errorf("expected 1 upcoming, got %d", len(m.upcoming))
	}
	if len(m.undated) != 1 {
		t.Errorf("expected 1 undated, got %d", len(m.undated))
	}
	if len(m.allTasks) != 4 {
		t.Errorf("expected 4 allTasks, got %d", len(m.allTasks))
	}
}

func TestDayView_CursorClamp(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	// Add two tasks due soon (future, within today)
	due1 := now.Add(2 * time.Minute)
	due2 := now.Add(3 * time.Minute)
	t1, err := m.svc.Add("task one", &due1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.svc.Add("task two", &due2); err != nil {
		t.Fatal(err)
	}

	m.refreshData()

	if len(m.allTasks) < 2 {
		t.Fatalf("expected at least 2 tasks, got %d", len(m.allTasks))
	}

	m.cursor = len(m.allTasks) - 1

	if err := m.svc.Delete(t1.ID); err != nil {
		t.Fatal(err)
	}
	m.refreshData()

	if m.cursor >= len(m.allTasks) {
		t.Errorf("cursor %d should be < len(allTasks) %d", m.cursor, len(m.allTasks))
	}
}

func TestRenderDayView_Sections(t *testing.T) {
	m := testDayModel(t)
	now := time.Now()

	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 10, 0, 0, 0, time.Local)
	m.svc.Add("overdue task", &yesterday)

	todaySoon := now.Add(1 * time.Minute)
	m.svc.Add("today task", &todaySoon)

	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 10, 0, 0, 0, time.Local)
	m.svc.Add("upcoming task", &tomorrow)

	m.svc.Add("undated task", nil)
	m.refreshData()

	output := renderDayView(m)
	for _, section := range []string{"OVERDUE", "TODAY", "UPCOMING", "ANYTIME"} {
		if !strings.Contains(output, section) {
			t.Errorf("expected output to contain %q, got:\n%s", section, output)
		}
	}
}

func TestRenderDayView_Empty(t *testing.T) {
	m := testDayModel(t)
	output := renderDayView(m)
	if !strings.Contains(output, "No tasks") {
		t.Errorf("expected output to contain 'No tasks', got:\n%s", output)
	}
}

func TestRenderTask_CursorPrefix(t *testing.T) {
	now := time.Now()
	tk := task.Task{
		ID:        "abcdef1234567890",
		Title:     "Test task",
		CreatedAt: now,
	}

	selected := renderTask(tk, 0, 0)
	if !strings.Contains(selected, ">") {
		t.Errorf("selected task should contain '>', got:\n%s", selected)
	}

	notSelected := renderTask(tk, 1, 0)
	if strings.Contains(notSelected, ">") {
		t.Errorf("non-selected task should not contain '>', got:\n%s", notSelected)
	}
}

func TestDayView_EmptyState(t *testing.T) {
	m := testDayModel(t)

	if m.cursor != 0 {
		t.Errorf("cursor should be 0 with no tasks, got %d", m.cursor)
	}
	if len(m.allTasks) != 0 {
		t.Errorf("allTasks should be empty, got %d", len(m.allTasks))
	}
}
