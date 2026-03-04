package task

import "time"

const defaultLeadTime = 15 * time.Minute

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
	ListDueReminders() ([]Task, error)
	MarkReminded(id string) error
}

// Service provides business logic for task management.
type Service struct {
	repo     Repository
	leadTime time.Duration
}

// NewService creates a new task service with default reminder lead time.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, leadTime: defaultLeadTime}
}

// SetLeadTime configures the reminder lead time.
func (svc *Service) SetLeadTime(d time.Duration) {
	svc.leadTime = d
}

// Add creates a new task with an optional due date.
// If a due date is provided, RemindAt is auto-calculated based on lead time.
func (svc *Service) Add(title string, dueAt *time.Time) (Task, error) {
	t := Task{
		ID:        NewID(),
		Title:     title,
		DueAt:     dueAt,
		CreatedAt: time.Now(),
	}

	if dueAt != nil {
		remind := dueAt.Add(-svc.leadTime)
		t.RemindAt = &remind
	}

	if err := svc.repo.Create(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// AddRecurring creates a recurring task with pattern and time.
func (svc *Service) AddRecurring(title, pattern, timeStr string) (Task, error) {
	nextDue, err := NextOccurrence(pattern, timeStr, time.Now())
	if err != nil {
		return Task{}, err
	}

	remind := nextDue.Add(-svc.leadTime)
	t := Task{
		ID:                NewID(),
		Title:             title,
		DueAt:             &nextDue,
		RemindAt:          &remind,
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

	remind := nextDue.Add(-svc.leadTime)
	t.DueAt = &nextDue
	t.RemindAt = &remind
	t.Reminded = false
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

	remind := newDue.Add(-svc.leadTime)
	t.DueAt = &newDue
	t.RemindAt = &remind
	t.Reminded = false
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

// ListByDateRange returns tasks due within [start, end).
func (svc *Service) ListByDateRange(start, end time.Time) ([]Task, error) {
	return svc.repo.ListByDateRange(start, end)
}

// Get retrieves a task by ID.
func (svc *Service) Get(id string) (Task, error) {
	return svc.repo.Get(id)
}

// GetDueReminders returns tasks that need reminders right now.
func (svc *Service) GetDueReminders() ([]Task, error) {
	return svc.repo.ListDueReminders()
}

// MarkReminded marks a task's reminder as fired.
func (svc *Service) MarkReminded(id string) error {
	return svc.repo.MarkReminded(id)
}

// Update saves changes to a task.
func (svc *Service) Update(t Task) error {
	return svc.repo.Update(t)
}
