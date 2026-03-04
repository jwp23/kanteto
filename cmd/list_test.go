package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func setupTestService(t *testing.T) *task.Service {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return task.NewService(s)
}

func execList(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	// Reset flags between test runs
	listNext = false
	listPrev = false
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"list"}, args...))
	// Bypass PersistentPreRunE since we set svc manually
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestListNext(t *testing.T) {
	testSvc := setupTestService(t)

	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrowNoon := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, tomorrow.Location())
	testSvc.Add("Tomorrow task", &tomorrowNoon)

	out, err := execList(t, testSvc, "--next")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Tomorrow task") {
		t.Errorf("--next should show tomorrow's tasks, got:\n%s", out)
	}
}

func TestListPrev(t *testing.T) {
	testSvc := setupTestService(t)

	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayNoon := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 12, 0, 0, 0, yesterday.Location())
	testSvc.Add("Yesterday task", &yesterdayNoon)

	out, err := execList(t, testSvc, "--prev")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Yesterday task") {
		t.Errorf("--prev should show yesterday's tasks, got:\n%s", out)
	}
}

func TestListNextAndPrevMutuallyExclusive(t *testing.T) {
	testSvc := setupTestService(t)
	_, err := execList(t, testSvc, "--next", "--prev")
	if err == nil {
		t.Error("expected error when both --next and --prev are set")
	}
}

func TestListDefaultShowsToday(t *testing.T) {
	testSvc := setupTestService(t)

	// Use a time in the future today to avoid overdue
	now := time.Now()
	todayLater := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, now.Location())
	testSvc.Add("Today task", &todayLater)

	// Add tomorrow's task — should appear in UPCOMING, not TODAY
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowNoon := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, tomorrow.Location())
	testSvc.Add("Tomorrow task", &tomorrowNoon)

	out, err := execList(t, testSvc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TODAY") {
		t.Errorf("default list should have TODAY section, got:\n%s", out)
	}
	if !strings.Contains(out, "Today task") {
		t.Errorf("default list should show today's tasks, got:\n%s", out)
	}
}

func TestListPrevShowsYesterdayInToday(t *testing.T) {
	testSvc := setupTestService(t)

	// Task due yesterday at noon — with --prev, it should appear in TODAY section
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayNoon := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 12, 0, 0, 0, yesterday.Location())
	testSvc.Add("Yesterday task", &yesterdayNoon)

	out, err := execList(t, testSvc, "--prev")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TODAY") {
		t.Errorf("--prev should show yesterday's tasks under TODAY, got:\n%s", out)
	}
}
