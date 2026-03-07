package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execEdit(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	// Reset flags
	editTitle = ""
	editBy = ""
	editEvery = ""

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"edit"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

	// Capture stdout because edit.go uses fmt.Printf
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

func TestEdit_Title(t *testing.T) {
	testSvc := setupTestService(t)

	tk, err := testSvc.Add("Original title", nil)
	if err != nil {
		t.Fatal(err)
	}

	prefix := tk.ID[:8]
	out, err := execEdit(t, testSvc, prefix, "--title", "New title")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Updated: New title") {
		t.Errorf("expected output to contain 'Updated: New title', got:\n%s", out)
	}
}

func TestEdit_NotFound(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execEdit(t, testSvc, "zzzzzzzz", "--title", "Nope")
	if err == nil {
		t.Error("expected error for bogus task ID")
	}
}
