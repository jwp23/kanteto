package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/store"
	"github.com/jwp23/kanteto/internal/task"
)

func setupProfileTest(t *testing.T) (*task.Service, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	ps := store.NewProfileStore(s, "default")
	svc := task.NewService(ps)
	return svc, s
}

func execProfile(t *testing.T, testSvc *task.Service, testStore *store.Store, args ...string) (string, error) {
	t.Helper()
	svc = testSvc
	rawStore = testStore
	cfg = config.Config{ActiveProfile: "default", Backend: "sqlite"}
	profileOverride = ""

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"profile"}, args...))
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

func TestProfileShow(t *testing.T) {
	testSvc, testStore := setupProfileTest(t)

	out, err := execProfile(t, testSvc, testStore, "show")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "default") {
		t.Errorf("expected 'default', got:\n%s", out)
	}
}

func TestProfileList(t *testing.T) {
	testSvc, testStore := setupProfileTest(t)

	// Add tasks with different profiles
	testStore.Create(task.Task{ID: task.NewID(), Title: "Work", Profile: "work"})
	testStore.Create(task.Task{ID: task.NewID(), Title: "Personal", Profile: "personal"})

	out, err := execProfile(t, testSvc, testStore, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "personal") {
		t.Errorf("expected 'personal' in output, got:\n%s", out)
	}
}

func TestProfileListShowsActiveMark(t *testing.T) {
	testSvc, testStore := setupProfileTest(t)

	testStore.Create(task.Task{ID: task.NewID(), Title: "Default task", Profile: "default"})
	testStore.Create(task.Task{ID: task.NewID(), Title: "Work task", Profile: "work"})

	out, err := execProfile(t, testSvc, testStore, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "* default") {
		t.Errorf("expected '* default' (active marker), got:\n%s", out)
	}
	if !strings.Contains(out, "  work") {
		t.Errorf("expected '  work' (not active), got:\n%s", out)
	}
}
