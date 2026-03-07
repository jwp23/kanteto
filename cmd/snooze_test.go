package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jwp23/kanteto/internal/task"
)

func execSnooze(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	// Reset flags
	snoozeFor = "1h"

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"snooze"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

	// Capture stdout because snooze.go uses fmt.Printf
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

func TestSnooze_HappyPath(t *testing.T) {
	testSvc := setupTestService(t)

	due := time.Now().Add(2 * time.Hour)
	tk, err := testSvc.Add("Review PR", &due)
	if err != nil {
		t.Fatal(err)
	}

	prefix := tk.ID[:8]
	out, err := execSnooze(t, testSvc, prefix, "--for", "1h")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Snoozed:") {
		t.Errorf("expected output to contain 'Snoozed:', got:\n%s", out)
	}
}

func TestSnooze_NotFound(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execSnooze(t, testSvc, "zzzzzzzz", "--for", "1h")
	if err == nil {
		t.Error("expected error for bogus task ID")
	}
}
