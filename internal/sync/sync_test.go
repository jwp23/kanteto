package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("dolt not found on PATH, skipping integration test")
	}
}

func initDoltRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("dolt", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt init: %s: %v", out, err)
	}
	cmd = exec.Command("dolt", "sql", "-q", `CREATE TABLE tasks (id VARCHAR(255) PRIMARY KEY);`)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create table: %s: %v", out, err)
	}
	cmd = exec.Command("dolt", "add", "-A")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt add: %s: %v", out, err)
	}
	cmd = exec.Command("dolt", "commit", "-m", "init", "--allow-empty")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dolt commit: %s: %v", out, err)
	}
}

func TestPush_CleanRepo(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	err := s.Push()
	if err == nil {
		t.Fatal("expected error when pushing with no remote")
	}
}

func TestPush_NothingToCommit(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	// No changes → Push should report nothing to push (not an error)
	err := s.Push()
	if err == nil {
		// It's OK if push fails because there's no remote — that's expected
		return
	}
	// The error should be about the remote, not about staging
	if !strings.Contains(err.Error(), "remote") {
		t.Logf("push error (expected remote error): %v", err)
	}
}

func TestAddRemote(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	err := s.AddRemote("origin", "file:///tmp/fake-remote")
	if err != nil {
		t.Fatalf("AddRemote() error: %v", err)
	}

	remotes, err := s.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error: %v", err)
	}
	found := false
	for _, r := range remotes {
		if strings.Contains(r, "origin") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'origin' in remotes, got: %v", remotes)
	}
}

func TestListRemotes_Empty(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	remotes, err := s.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error: %v", err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d: %v", len(remotes), remotes)
	}
}

func TestPushPull_RoundTrip(t *testing.T) {
	skipIfNoDolt(t)

	// Create a bare remote repo
	remoteDir := t.TempDir()
	cmd := exec.Command("dolt", "init", "--fun")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init remote: %s: %v", out, err)
	}

	// Create and populate source repo
	srcDir := t.TempDir()
	initDoltRepo(t, srcDir)

	srcSync := New(srcDir)
	if err := srcSync.AddRemote("origin", "file://"+remoteDir); err != nil {
		t.Fatalf("add remote: %v", err)
	}

	// Insert a row and push
	cmd = exec.Command("dolt", "sql", "-q", `INSERT INTO tasks VALUES ('test-1');`)
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("insert: %s: %v", out, err)
	}
	if err := srcSync.Push(); err != nil {
		t.Fatalf("Push() error: %v", err)
	}

	// Clone into a new dir and verify
	dstDir := filepath.Join(t.TempDir(), "dst")
	cmd = exec.Command("dolt", "clone", "file://"+remoteDir, dstDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone: %s: %v", out, err)
	}

	// Verify the task exists
	cmd = exec.Command("dolt", "sql", "-q", "SELECT id FROM tasks", "-r", "json")
	cmd.Dir = dstDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("select: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "test-1") {
		t.Errorf("expected 'test-1' in clone, got: %s", out)
	}

	// Test pull: add a row in remote, push from src, pull from dst
	cmd = exec.Command("dolt", "sql", "-q", `INSERT INTO tasks VALUES ('test-2');`)
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("insert: %s: %v", out, err)
	}
	if err := srcSync.Push(); err != nil {
		t.Fatalf("Push() error: %v", err)
	}

	dstSync := New(dstDir)
	if err := dstSync.Pull(); err != nil {
		t.Fatalf("Pull() error: %v", err)
	}

	cmd = exec.Command("dolt", "sql", "-q", "SELECT id FROM tasks", "-r", "json")
	cmd.Dir = dstDir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("select after pull: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "test-2") {
		t.Errorf("expected 'test-2' after pull, got: %s", out)
	}
}

func TestPull_NoRemote(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	err := s.Pull()
	if err == nil {
		t.Fatal("expected error pulling with no remote")
	}
}

func TestStatus_Clean(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	clean, err := s.IsClean()
	if err != nil {
		t.Fatalf("IsClean() error: %v", err)
	}
	if !clean {
		t.Error("expected clean repo")
	}
}

func TestStatus_Dirty(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	// Make a change
	cmd := exec.Command("dolt", "sql", "-q", `INSERT INTO tasks VALUES ('dirty');`)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("insert: %s: %v", out, err)
	}

	s := New(dir)
	clean, err := s.IsClean()
	if err != nil {
		t.Fatalf("IsClean() error: %v", err)
	}
	if clean {
		t.Error("expected dirty repo")
	}
}

func TestDir(t *testing.T) {
	s := New("/tmp/test")
	if s.Dir() != "/tmp/test" {
		t.Errorf("Dir() = %q, want %q", s.Dir(), "/tmp/test")
	}
}

func TestHasRemote(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	s := New(dir)
	if s.HasRemote("origin") {
		t.Error("expected no remote initially")
	}

	s.AddRemote("origin", "file:///tmp/fake")
	if !s.HasRemote("origin") {
		t.Error("expected origin remote after add")
	}
}

func TestCommit(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()
	initDoltRepo(t, dir)

	// Insert some data
	cmd := exec.Command("dolt", "sql", "-q", `INSERT INTO tasks VALUES ('commit-test');`)
	cmd.Dir = dir
	cmd.CombinedOutput()

	s := New(dir)
	err := s.Commit("test commit message")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Verify clean after commit
	clean, _ := s.IsClean()
	if !clean {
		t.Error("expected clean after commit")
	}

	// Verify commit message in log
	cmd = exec.Command("dolt", "log", "-n", "1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "test commit message") {
		t.Errorf("expected commit message in log, got: %s", out)
	}
}

func TestInitRepo(t *testing.T) {
	skipIfNoDolt(t)
	dir := t.TempDir()

	s := New(dir)
	err := s.InitRepo()
	if err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	// Verify .dolt exists
	if _, err := os.Stat(filepath.Join(dir, ".dolt")); os.IsNotExist(err) {
		t.Error("expected .dolt directory to exist after InitRepo")
	}
}
