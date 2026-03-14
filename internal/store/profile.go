package store

import (
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

// ProfileStore wraps a task.Repository and scopes all operations to a profile.
type ProfileStore struct {
	inner   task.Repository
	profile string
}

// NewProfileStore creates a ProfileStore that filters by the given profile.
func NewProfileStore(inner task.Repository, profile string) *ProfileStore {
	return &ProfileStore{inner: inner, profile: profile}
}

func (ps *ProfileStore) Create(t task.Task) error {
	t.Profile = ps.profile
	return ps.inner.Create(t)
}

func (ps *ProfileStore) Get(id string) (task.Task, error) {
	return ps.inner.Get(id)
}

func (ps *ProfileStore) Complete(id string) error {
	return ps.inner.Complete(id)
}

func (ps *ProfileStore) Delete(id string) error {
	return ps.inner.Delete(id)
}

func (ps *ProfileStore) Update(t task.Task) error {
	return ps.inner.Update(t)
}

func (ps *ProfileStore) UpdateDueAt(id string, dueAt *time.Time) error {
	return ps.inner.UpdateDueAt(id, dueAt)
}

func (ps *ProfileStore) ListAll(includeCompleted bool) ([]task.Task, error) {
	all, err := ps.inner.ListAll(includeCompleted)
	if err != nil {
		return nil, err
	}
	return filterProfile(all, ps.profile), nil
}

func (ps *ProfileStore) ListByDateRange(start, end time.Time) ([]task.Task, error) {
	all, err := ps.inner.ListByDateRange(start, end)
	if err != nil {
		return nil, err
	}
	return filterProfile(all, ps.profile), nil
}

func (ps *ProfileStore) ListOverdue() ([]task.Task, error) {
	all, err := ps.inner.ListOverdue()
	if err != nil {
		return nil, err
	}
	return filterProfile(all, ps.profile), nil
}

func (ps *ProfileStore) ListOverdueAsOf(asOf time.Time) ([]task.Task, error) {
	all, err := ps.inner.ListOverdueAsOf(asOf)
	if err != nil {
		return nil, err
	}
	return filterProfile(all, ps.profile), nil
}

func (ps *ProfileStore) ListUndated() ([]task.Task, error) {
	all, err := ps.inner.ListUndated()
	if err != nil {
		return nil, err
	}
	return filterProfile(all, ps.profile), nil
}

func (ps *ProfileStore) ListProfiles() ([]string, error) {
	return ps.inner.ListProfiles()
}

func filterProfile(tasks []task.Task, profile string) []task.Task {
	var filtered []task.Task
	for _, t := range tasks {
		if t.Profile == profile {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
