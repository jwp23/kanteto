package task

import (
	"time"

	"github.com/jwp23/kanteto/internal/nlp"
)

// Repository defines the storage contract for tasks.
type Repository interface {
	Create(t Task) error
	Get(id string) (Task, error)
	Complete(id string) error
	Delete(id string) error
	Update(t Task) error
	UpdateDueAt(id string, dueAt *time.Time) error
	ListAll(includeCompleted bool) ([]Task, error)
	ListByDateRange(start, end time.Time) ([]Task, error)
	ListOverdue() ([]Task, error)
	ListOverdueAsOf(asOf time.Time) ([]Task, error)
	ListUndated() ([]Task, error)
	ListProfiles() ([]string, error)
}

// Service provides business logic for task management.
type Service struct {
	repo Repository
}

// NewService creates a new task service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Add creates a new task with an optional due date and tags.
func (svc *Service) Add(title string, dueAt *time.Time, tags ...string) (Task, error) {
	if tags == nil {
		tags = []string{}
	}
	t := Task{
		ID:        NewID(),
		Title:     title,
		DueAt:     dueAt,
		Tags:      tags,
		CreatedAt: time.Now(),
	}

	if err := svc.repo.Create(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// AddTag adds a tag to a task. Duplicate tags are ignored.
func (svc *Service) AddTag(id, tag string) error {
	t, err := svc.repo.Get(id)
	if err != nil {
		return err
	}
	for _, existing := range t.Tags {
		if existing == tag {
			return nil
		}
	}
	t.Tags = append(t.Tags, tag)
	return svc.repo.Update(t)
}

// RemoveTag removes a tag from a task. Missing tags are ignored.
func (svc *Service) RemoveTag(id, tag string) error {
	t, err := svc.repo.Get(id)
	if err != nil {
		return err
	}
	filtered := t.Tags[:0]
	for _, existing := range t.Tags {
		if existing != tag {
			filtered = append(filtered, existing)
		}
	}
	t.Tags = filtered
	return svc.repo.Update(t)
}

// AddRecurring creates a recurring task with pattern and time.
func (svc *Service) AddRecurring(title, pattern, timeStr string) (Task, error) {
	nextDue, err := NextOccurrence(pattern, timeStr, time.Now())
	if err != nil {
		return Task{}, err
	}

	t := Task{
		ID:                NewID(),
		Title:             title,
		DueAt:             &nextDue,
		CreatedAt:         time.Now(),
		RecurrencePattern: pattern,
		RecurrenceTime:    timeStr,
	}

	if err := svc.repo.Create(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Complete marks a task as done. For recurring tasks, it advances to
// the next occurrence instead of marking complete.
func (svc *Service) Complete(id string) error {
	t, err := svc.repo.Get(id)
	if err != nil {
		return err
	}

	// Non-recurring: just mark complete
	if t.RecurrencePattern == "" {
		return svc.repo.Complete(id)
	}

	// Recurring: advance to next occurrence
	nextDue, err := NextOccurrence(t.RecurrencePattern, t.RecurrenceTime, time.Now())
	if err != nil {
		return err
	}

	t.DueAt = &nextDue
	return svc.repo.Update(t)
}

// Delete removes a task permanently.
func (svc *Service) Delete(id string) error {
	return svc.repo.Delete(id)
}

// Snooze postpones a task's due date by the given duration.
func (svc *Service) Snooze(id string, d time.Duration) error {
	t, err := svc.repo.Get(id)
	if err != nil {
		return err
	}

	var newDue time.Time
	if t.DueAt != nil {
		newDue = t.DueAt.Add(d)
	} else {
		newDue = time.Now().Add(d)
	}

	t.DueAt = &newDue
	return svc.repo.Update(t)
}

// ListAll returns all incomplete tasks.
func (svc *Service) ListAll() ([]Task, error) {
	return svc.repo.ListAll(false)
}

// ListToday returns incomplete tasks due today.
func (svc *Service) ListToday() ([]Task, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	return svc.repo.ListByDateRange(start, end)
}

// ListOverdue returns tasks that are past due and incomplete.
func (svc *Service) ListOverdue() ([]Task, error) {
	return svc.repo.ListOverdue()
}

// ListOverdueAsOf returns incomplete tasks with due dates before the given time.
func (svc *Service) ListOverdueAsOf(asOf time.Time) ([]Task, error) {
	return svc.repo.ListOverdueAsOf(asOf)
}

// ListUndated returns incomplete tasks with no due date.
func (svc *Service) ListUndated() ([]Task, error) {
	return svc.repo.ListUndated()
}

// ListByDateRange returns tasks due within [start, end).
func (svc *Service) ListByDateRange(start, end time.Time) ([]Task, error) {
	return svc.repo.ListByDateRange(start, end)
}

// Get retrieves a task by ID.
func (svc *Service) Get(id string) (Task, error) {
	return svc.repo.Get(id)
}

// SetDueAt updates a task's deadline.
func (svc *Service) SetDueAt(id string, dueAt time.Time) error {
	t, err := svc.repo.Get(id)
	if err != nil {
		return err
	}
	t.DueAt = &dueAt
	return svc.repo.Update(t)
}

// ReparseResult holds the outcome of a reparse operation.
type ReparseResult struct {
	Updated int
	Total   int
}

// Reparse scans undated tasks for inline deadlines, updating title and DueAt
// for any matches found.
func (svc *Service) Reparse() (ReparseResult, error) {
	undated, err := svc.repo.ListUndated()
	if err != nil {
		return ReparseResult{}, err
	}

	var result ReparseResult
	result.Total = len(undated)

	for _, t := range undated {
		title, dueAt := nlp.ExtractDeadline(t.Title)
		if dueAt != nil {
			t.Title = title
			t.DueAt = dueAt
			if err := svc.repo.Update(t); err != nil {
				return result, err
			}
			result.Updated++
		}
	}

	return result, nil
}

// Update saves changes to a task.
func (svc *Service) Update(t Task) error {
	return svc.repo.Update(t)
}
