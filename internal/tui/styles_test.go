package tui

import (
	"testing"
	"time"
)

func TestUrgencyColor(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		dueAt    *time.Time
		expected string
	}{
		{"nil due date", nil, string(urgencyWhite)},
		{"far future", ptrTime(now.Add(5 * time.Hour)), string(urgencyWhite)},
		{"2 hours", ptrTime(now.Add(90 * time.Minute)), string(urgencyYellow)},
		{"1 hour", ptrTime(now.Add(45 * time.Minute)), string(urgencyAmber)},
		{"30 min", ptrTime(now.Add(15 * time.Minute)), string(urgencyOrange)},
		{"overdue", ptrTime(now.Add(-time.Hour)), string(urgencyRed)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(UrgencyColor(tt.dueAt))
			if got != tt.expected {
				t.Errorf("UrgencyColor() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
