package nlp

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	// Fix "now" for deterministic tests
	now := time.Date(2026, 3, 3, 14, 0, 0, 0, time.Local)

	tests := []struct {
		input    string
		expected time.Time
	}{
		{"march 11", time.Date(2026, 3, 11, 23, 59, 0, 0, time.Local)},
		{"march 11 at 3pm", time.Date(2026, 3, 11, 15, 0, 0, 0, time.Local)},
		{"tomorrow", time.Date(2026, 3, 4, 23, 59, 0, 0, time.Local)},
		{"tomorrow at 9am", time.Date(2026, 3, 4, 9, 0, 0, 0, time.Local)},
		{"today at 5pm", time.Date(2026, 3, 3, 17, 0, 0, 0, time.Local)},
		{"next friday", time.Date(2026, 3, 6, 23, 59, 0, 0, time.Local)},
		{"2026-03-15", time.Date(2026, 3, 15, 23, 59, 0, 0, time.Local)},
		{"3/15", time.Date(2026, 3, 15, 23, 59, 0, 0, time.Local)},
		{"in 5 minutes", now.Add(5 * time.Minute)},
		{"in 1 hour", now.Add(time.Hour)},
		{"in 2 hours", now.Add(2 * time.Hour)},
		{"friday", time.Date(2026, 3, 6, 23, 59, 0, 0, time.Local)},
		{"friday at 2pm", time.Date(2026, 3, 6, 14, 0, 0, 0, time.Local)},
		{"this friday", time.Date(2026, 3, 6, 23, 59, 0, 0, time.Local)},
		{"monday", time.Date(2026, 3, 9, 23, 59, 0, 0, time.Local)},
		{"this mon", time.Date(2026, 3, 9, 23, 59, 0, 0, time.Local)},
		// Bare times without "at" keyword (kanteto-ejz.1)
		{"3pm", time.Date(2026, 3, 3, 15, 0, 0, 0, time.Local)},
		{"today 4pm", time.Date(2026, 3, 3, 16, 0, 0, 0, time.Local)},
		{"tomorrow 9am", time.Date(2026, 3, 4, 9, 0, 0, 0, time.Local)},
		{"friday 12pm", time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)},
		{"march 10 5pm", time.Date(2026, 3, 10, 17, 0, 0, 0, time.Local)},
		{"12am", time.Date(2026, 3, 3, 0, 0, 0, 0, time.Local)},
		{"12pm", time.Date(2026, 3, 3, 12, 0, 0, 0, time.Local)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDateRelativeTo(tt.input, now)
			if err != nil {
				t.Fatalf("ParseDateRelativeTo(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("ParseDateRelativeTo(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1 hour", time.Hour},
		{"30 minutes", 30 * time.Minute},
		{"2 hours", 2 * time.Hour},
		{"1 day", 24 * time.Hour},
		{"30m", 30 * time.Minute},
		{"2h", 2 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDate_Invalid(t *testing.T) {
	now := time.Now()
	_, err := ParseDateRelativeTo("not a date at all xyz", now)
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}

func TestExtractDeadline(t *testing.T) {
	tests := []struct {
		input     string
		wantTitle string
		wantDue   bool
	}{
		{"test kt in 5 minutes", "test kt", true},
		{"call dentist by tomorrow", "call dentist", true},
		{"buy groceries", "buy groceries", false},
		{"meeting at 3pm", "meeting", true},
		{"just a title", "just a title", false},
		// Trailing date expressions without markers (kanteto-ejz.2)
		{"review doc Friday 12pm", "review doc", true},
		{"review doc friday", "review doc", true},
		{"call mom tomorrow 9am", "call mom", true},
		{"fix bug march 10 5pm", "fix bug", true},
		{"friday", "friday", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			title, dueAt := ExtractDeadline(tt.input)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if (dueAt != nil) != tt.wantDue {
				t.Errorf("dueAt nil = %v, want hasDue = %v", dueAt == nil, tt.wantDue)
			}
		})
	}
}
