package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execAdd(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	// Reset flags
	addBy = ""
	addEvery = ""

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"add"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

	// Capture stdout because add.go uses fmt.Printf
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = origStdout

	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)
	combined := buf.String() + stdoutBuf.String()
	return combined, err
}

func TestAdd_HappyPath(t *testing.T) {
	testSvc := setupTestService(t)

	out, err := execAdd(t, testSvc, "Test task")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added: Test task") {
		t.Errorf("expected output to contain 'Added: Test task', got:\n%s", out)
	}
}

func TestAdd_WithDeadline(t *testing.T) {
	testSvc := setupTestService(t)

	out, err := execAdd(t, testSvc, "Test", "--by", "tomorrow")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "due") {
		t.Errorf("expected output to contain 'due', got:\n%s", out)
	}
}

func TestAdd_Recurring(t *testing.T) {
	testSvc := setupTestService(t)

	out, err := execAdd(t, testSvc, "Standup", "--every", "daily at 9am")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "recurring") {
		t.Errorf("expected output to contain 'recurring', got:\n%s", out)
	}
}

func TestAdd_Recurring_InvalidPattern(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execAdd(t, testSvc, "Bad", "--every", "gibberish")
	if err == nil {
		t.Error("expected error for invalid recurrence pattern")
	}
}

func TestAdd_MissingTitle(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execAdd(t, testSvc)
	if err == nil {
		t.Error("expected error when no title is provided")
	}
}
