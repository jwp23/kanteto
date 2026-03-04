package task

import (
	"testing"
	"time"
)

func TestParseRecurrence(t *testing.T) {
	tests := []struct {
		pattern string
		timeStr string
		from    time.Time
		wantErr bool
	}{
		{"daily", "9am", time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local), false},
		{"weekdays", "4pm", time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local), false},
		{"weekly", "5pm", time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local), false},
		{"friday", "5pm", time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local), false},
		{"invalid", "9am", time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local), true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.timeStr, func(t *testing.T) {
			_, err := NextOccurrence(tt.pattern, tt.timeStr, tt.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextOccurrence() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNextOccurrence_Daily(t *testing.T) {
	from := time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local)
	next, err := NextOccurrence("daily", "9am", from)
	if err != nil {
		t.Fatal(err)
	}

	expected := time.Date(2026, 3, 4, 9, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("got %v, want %v", next, expected)
	}
}

func TestNextOccurrence_Weekdays(t *testing.T) {
	// Friday -> should skip to Monday
	friday := time.Date(2026, 3, 6, 17, 0, 0, 0, time.Local)
	next, err := NextOccurrence("weekdays", "4pm", friday)
	if err != nil {
		t.Fatal(err)
	}

	monday := time.Date(2026, 3, 9, 16, 0, 0, 0, time.Local)
	if !next.Equal(monday) {
		t.Errorf("got %v, want %v (Monday)", next, monday)
	}
}

func TestNextOccurrence_SpecificDay(t *testing.T) {
	// Tuesday -> next Friday
	tuesday := time.Date(2026, 3, 3, 10, 0, 0, 0, time.Local)
	next, err := NextOccurrence("friday", "5pm", tuesday)
	if err != nil {
		t.Fatal(err)
	}

	expected := time.Date(2026, 3, 6, 17, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("got %v, want %v", next, expected)
	}
}

func TestParseRecurrenceSpec(t *testing.T) {
	tests := []struct {
		input   string
		pattern string
		timeStr string
		wantErr bool
	}{
		{"weekdays at 4pm", "weekdays", "4pm", false},
		{"friday at 5pm", "friday", "5pm", false},
		{"daily at 9am", "daily", "9am", false},
		{"every day at 9am", "daily", "9am", false},
		{"weekly at 5pm", "weekly", "5pm", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pattern, timeStr, err := ParseRecurrenceSpec(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRecurrenceSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if pattern != tt.pattern {
				t.Errorf("pattern = %q, want %q", pattern, tt.pattern)
			}
			if timeStr != tt.timeStr {
				t.Errorf("timeStr = %q, want %q", timeStr, tt.timeStr)
			}
		})
	}
}
