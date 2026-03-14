package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not found on PATH, skipping integration test")
	}
}

func execSync(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	// Override DataDir via XDG env var
	t.Setenv("XDG_DATA_HOME", dataDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"sync"}, args...))
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

func setupDoltRepo(t *testing.T) string {
	t.Helper()
	// DataDir returns XDG_DATA_HOME/kanteto, so we set parent
	parentDir := t.TempDir()
	dataDir := parentDir + "/kanteto"
	os.MkdirAll(dataDir, 0o755)

	cmd := exec.Command("dolt", "init")
	cmd.Dir = dataDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	cmd = exec.Command("dolt", "sql", "-q", `CREATE TABLE tasks (id VARCHAR(255) PRIMARY KEY);`)
	cmd.Dir = dataDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create table: %s: %v", out, err)
	}
	cmd = exec.Command("dolt", "add", "-A")
	cmd.Dir = dataDir
	cmd.CombinedOutput()
	cmd = exec.Command("dolt", "commit", "-m", "init", "--allow-empty")
	cmd.Dir = dataDir
	cmd.CombinedOutput()

	return parentDir
}

func TestSyncRemoteList_Empty(t *testing.T) {
	skipIfNoDolt(t)
	dataDir := setupDoltRepo(t)

	out, err := execSync(t, dataDir, "remote", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No remotes") {
		t.Errorf("expected 'No remotes', got:\n%s", out)
	}
}

func TestSyncRemoteAdd(t *testing.T) {
	skipIfNoDolt(t)
	dataDir := setupDoltRepo(t)

	out, err := execSync(t, dataDir, "remote", "add", "origin", "file:///tmp/fake")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added remote") {
		t.Errorf("expected 'Added remote', got:\n%s", out)
	}

	// Verify it shows up in list
	out, err = execSync(t, dataDir, "remote", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "origin") {
		t.Errorf("expected 'origin' in list, got:\n%s", out)
	}
}

func TestSyncPush_NoRemote(t *testing.T) {
	skipIfNoDolt(t)
	dataDir := setupDoltRepo(t)

	_, err := execSync(t, dataDir, "push")
	if err == nil {
		t.Error("expected error when pushing with no remote")
	}
}

func TestSyncPull_NoRemote(t *testing.T) {
	skipIfNoDolt(t)
	dataDir := setupDoltRepo(t)

	_, err := execSync(t, dataDir, "pull")
	if err == nil {
		t.Error("expected error when pulling with no remote")
	}
}
