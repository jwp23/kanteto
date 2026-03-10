package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execReparse(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	reparseApply = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"reparse"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestReparse_DryRun(t *testing.T) {
	testSvc := setupTestService(t)
	testSvc.Add("review doc friday 12pm", nil)

	out, err := execReparse(t, testSvc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "review doc") {
		t.Errorf("expected proposed change in output, got:\n%s", out)
	}
	if !strings.Contains(out, "would be updated") {
		t.Errorf("expected dry-run message, got:\n%s", out)
	}

	// Verify DB unchanged
	tasks, err := testSvc.ListUndated()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 undated task, got %d", len(tasks))
	}
	if tasks[0].Title != "review doc friday 12pm" {
		t.Errorf("title should be unchanged, got %q", tasks[0].Title)
	}
}

func TestReparse_Apply(t *testing.T) {
	testSvc := setupTestService(t)
	tk, err := testSvc.Add("call mom tomorrow 9am", nil)
	if err != nil {
		t.Fatal(err)
	}

	out, errExec := execReparse(t, testSvc, "--apply")
	if errExec != nil {
		t.Fatal(errExec)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("expected updated message, got:\n%s", out)
	}

	updated, err := testSvc.Get(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "call mom" {
		t.Errorf("expected title 'call mom', got %q", updated.Title)
	}
	if updated.DueAt == nil {
		t.Error("DueAt should not be nil after apply")
	}
}

func TestReparse_NoUndated(t *testing.T) {
	testSvc := setupTestService(t)

	out, err := execReparse(t, testSvc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No undated tasks found") {
		t.Errorf("expected no undated message, got:\n%s", out)
	}
}

func TestReparse_NoMatches(t *testing.T) {
	testSvc := setupTestService(t)
	testSvc.Add("buy groceries", nil)

	out, err := execReparse(t, testSvc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No deadlines detected") {
		t.Errorf("expected no-match message, got:\n%s", out)
	}
}
