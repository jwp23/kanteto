package task

import "time"

// Task represents a small commitment or promise to track.
type Task struct {
	ID                string
	Title             string
	DueAt             *time.Time
	Completed         bool
	CompletedAt       *time.Time
	CreatedAt         time.Time
	RecurrencePattern string
	RecurrenceTime string
	Tags           []string
	Profile           string
}

// IsOverdue returns true if the task has a past due date and is not completed.
func (t Task) IsOverdue() bool {
	if t.Completed || t.DueAt == nil {
		return false
	}
	return t.DueAt.Before(time.Now())
}

// IsDueToday returns true if the task is due sometime today.
func (t Task) IsDueToday() bool {
	if t.DueAt == nil {
		return false
	}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return !t.DueAt.Before(startOfDay) && t.DueAt.Before(endOfDay)
}
