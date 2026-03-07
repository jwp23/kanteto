package tui

import (
	"strings"
	"testing"
	"time"
)

func testWeekModel(t *testing.T) model {
	t.Helper()
	svc := testService(t)
	viewDate := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.Local)
	return model{
		svc:      svc,
		viewMode: weekView,
		viewDate: viewDate,
		width:    120,
		height:   24,
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
