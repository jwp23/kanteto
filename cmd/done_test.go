package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execDone(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"done"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

	// Capture stdout because done.go uses fmt.Printf
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

func TestDone_HappyPath(t *testing.T) {
	testSvc := setupTestService(t)

	tk, err := testSvc.Add("Finish report", nil)
	if err != nil {
		t.Fatal(err)
	}

	prefix := tk.ID[:8]
	out, err := execDone(t, testSvc, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Completed:") {
		t.Errorf("expected output to contain 'Completed:', got:\n%s", out)
	}
}

func TestDone_NotFound(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execDone(t, testSvc, "zzzzzzzz")
	if err == nil {
		t.Error("expected error for bogus task ID")
	}
}
