package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execRm(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"rm"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

	// Capture stdout because rm.go uses fmt.Printf
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

func TestRm_HappyPath(t *testing.T) {
	testSvc := setupTestService(t)

	tk, err := testSvc.Add("Old task", nil)
	if err != nil {
		t.Fatal(err)
	}

	prefix := tk.ID[:8]
	out, err := execRm(t, testSvc, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Deleted:") {
		t.Errorf("expected output to contain 'Deleted:', got:\n%s", out)
	}
}

func TestRm_NotFound(t *testing.T) {
	testSvc := setupTestService(t)

	_, err := execRm(t, testSvc, "zzzzzzzz")
	if err == nil {
		t.Error("expected error for bogus task ID")
	}
}
