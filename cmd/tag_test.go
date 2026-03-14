package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/task"
)

func execTag(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"tag"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

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

func execUntag(t *testing.T, testSvc *task.Service, args ...string) (string, error) {
	t.Helper()
	svc = testSvc

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"untag"}, args...))
	origPre := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	defer func() { rootCmd.PersistentPreRunE = origPre }()

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

func TestTag_AddTag(t *testing.T) {
	testSvc := setupTestService(t)

	tk, _ := testSvc.Add("Taggable task", nil)
	prefix := tk.ID[:8]

	out, err := execTag(t, testSvc, prefix, "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Tagged") {
		t.Errorf("expected output to contain 'Tagged', got:\n%s", out)
	}

	got, _ := testSvc.Get(tk.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "work" {
		t.Errorf("Tags = %v, want [work]", got.Tags)
	}
}

func TestUntag_RemoveTag(t *testing.T) {
	testSvc := setupTestService(t)

	tk, _ := testSvc.Add("Tagged task", nil, "work", "urgent")
	prefix := tk.ID[:8]

	out, err := execUntag(t, testSvc, prefix, "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Untagged") {
		t.Errorf("expected output to contain 'Untagged', got:\n%s", out)
	}

	got, _ := testSvc.Get(tk.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("Tags = %v, want [urgent]", got.Tags)
	}
}
