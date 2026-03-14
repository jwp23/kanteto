package store

import (
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

func TestProfileStore_CreateSetsProfile(t *testing.T) {
	s := newTestStore(t)
	ps := NewProfileStore(s, "work")

	tk := task.Task{
		ID:        task.NewID(),
		Title:     "Work task",
		CreatedAt: time.Now(),
	}
	if err := ps.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "work" {
		t.Errorf("Profile = %q, want %q", got.Profile, "work")
	}
}

func TestProfileStore_ListAllFilters(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	s.Create(task.Task{ID: task.NewID(), Title: "Work task", CreatedAt: now, Profile: "work"})
	s.Create(task.Task{ID: task.NewID(), Title: "Personal task", CreatedAt: now, Profile: "personal"})
	s.Create(task.Task{ID: task.NewID(), Title: "Default task", CreatedAt: now, Profile: "default"})

	workStore := NewProfileStore(s, "work")
	tasks, err := workStore.ListAll(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 work task, got %d", len(tasks))
	}
	if tasks[0].Title != "Work task" {
		t.Errorf("Title = %q, want %q", tasks[0].Title, "Work task")
	}
}

func TestProfileStore_ListByDateRangeFilters(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	due := now.Add(1 * time.Hour)

	s.Create(task.Task{ID: task.NewID(), Title: "Work due", DueAt: &due, CreatedAt: now, Profile: "work"})
	s.Create(task.Task{ID: task.NewID(), Title: "Personal due", DueAt: &due, CreatedAt: now, Profile: "personal"})

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	workStore := NewProfileStore(s, "work")
	tasks, err := workStore.ListByDateRange(startOfDay, endOfDay)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Work due" {
		t.Errorf("Title = %q, want %q", tasks[0].Title, "Work due")
	}
}
