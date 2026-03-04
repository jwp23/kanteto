package task

import (
	"testing"
	"time"
)

func TestTask_IsOverdue(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "no due date is not overdue",
			task:     Task{Title: "test"},
			expected: false,
		},
		{
			name:     "future due date is not overdue",
			task:     Task{Title: "test", DueAt: ptrTime(now.Add(time.Hour))},
			expected: false,
		},
		{
			name:     "past due date is overdue",
			task:     Task{Title: "test", DueAt: ptrTime(now.Add(-time.Hour))},
			expected: true,
		},
		{
			name:     "completed task is not overdue",
			task:     Task{Title: "test", DueAt: ptrTime(now.Add(-time.Hour)), Completed: true},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsOverdue(); got != tt.expected {
				t.Errorf("IsOverdue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTask_IsDueToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "no due date is not due today",
			task:     Task{Title: "test"},
			expected: false,
		},
		{
			name:     "due later today is due today",
			task:     Task{Title: "test", DueAt: ptrTime(today)},
			expected: true,
		},
		{
			name:     "due tomorrow is not due today",
			task:     Task{Title: "test", DueAt: ptrTime(tomorrow)},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsDueToday(); got != tt.expected {
				t.Errorf("IsDueToday() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
