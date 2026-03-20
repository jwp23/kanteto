package sync

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/dolthub/driver"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dsn := fmt.Sprintf("file://%s?commitname=test&commitemail=test@test&database=testdb", dir)
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		t.Fatalf("open dolt: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// The embedded driver requires the database to be explicitly created.
	for _, q := range []string{
		"DROP DATABASE IF EXISTS testdb",
		"CREATE DATABASE testdb",
		"USE testdb",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("init db (%s): %v", q, err)
		}
	}
	return db
}

func TestInitRepo(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}
}

func TestIsClean_AfterInit(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	clean, err := s.IsClean()
	if err != nil {
		t.Fatalf("IsClean() error: %v", err)
	}
	if !clean {
		t.Error("expected clean repo after InitRepo")
	}
}

func TestIsClean_Dirty(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	// Create a table to make the working set dirty.
	if _, err := db.Exec("CREATE TABLE tasks (id VARCHAR(255) PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	clean, err := s.IsClean()
	if err != nil {
		t.Fatalf("IsClean() error: %v", err)
	}
	if clean {
		t.Error("expected dirty repo after creating a table")
	}
}

func TestCommit(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	// Create a table and insert data.
	if _, err := db.Exec("CREATE TABLE tasks (id VARCHAR(255) PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO tasks VALUES ('commit-test')"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := s.Commit("test commit message"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Verify working set is clean after commit.
	clean, err := s.IsClean()
	if err != nil {
		t.Fatalf("IsClean() error: %v", err)
	}
	if !clean {
		t.Error("expected clean repo after Commit")
	}

	// Verify the commit message appears in the log.
	var msg string
	err = db.QueryRow("SELECT message FROM dolt_log ORDER BY date DESC LIMIT 1").Scan(&msg)
	if err != nil {
		t.Fatalf("query dolt_log: %v", err)
	}
	if !strings.Contains(msg, "test commit message") {
		t.Errorf("expected commit message in log, got: %s", msg)
	}
}

func TestSnapshot_NothingToCommit(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	// Snapshot on a clean repo should succeed (no-op).
	if err := s.Snapshot("no changes"); err != nil {
		t.Fatalf("Snapshot() on clean repo error: %v", err)
	}
}

func TestAddRemote(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	if err := s.AddRemote("origin", "file:///tmp/fake-remote"); err != nil {
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
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	remotes, err := s.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error: %v", err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d: %v", len(remotes), remotes)
	}
}

func TestHasRemote(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	if s.HasRemote("origin") {
		t.Error("expected no remote initially")
	}

	if err := s.AddRemote("origin", "file:///tmp/fake"); err != nil {
		t.Fatalf("AddRemote() error: %v", err)
	}
	if !s.HasRemote("origin") {
		t.Error("expected origin remote after add")
	}
}

func TestPush_NoRemote(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	err := s.Push()
	if err == nil {
		t.Fatal("expected error when pushing with no remote")
	}
}

func TestPull_NoRemote(t *testing.T) {
	db := testDB(t)
	s := New(db)
	if err := s.InitRepo(); err != nil {
		t.Fatalf("InitRepo() error: %v", err)
	}

	err := s.Pull()
	if err == nil {
		t.Fatal("expected error pulling with no remote")
	}
}

func TestPushRemote_WithRemote(t *testing.T) {
	// Set up the "remote" database that will act as the remote store.
	remoteDir := t.TempDir()
	remoteDSN := fmt.Sprintf("file://%s?commitname=test&commitemail=test@test&database=testdb", remoteDir)
	remoteDB, err := sql.Open("dolt", remoteDSN)
	if err != nil {
		t.Fatalf("open remote dolt: %v", err)
	}
	for _, q := range []string{
		"DROP DATABASE IF EXISTS testdb",
		"CREATE DATABASE testdb",
		"USE testdb",
	} {
		if _, err := remoteDB.Exec(q); err != nil {
			t.Fatalf("init remote db (%s): %v", q, err)
		}
	}
	remoteSync := New(remoteDB)
	if err := remoteSync.InitRepo(); err != nil {
		t.Fatalf("remote InitRepo() error: %v", err)
	}
	remoteDB.Close()

	// Set up the source database.
	srcDB := testDB(t)
	srcSync := New(srcDB)
	if err := srcSync.InitRepo(); err != nil {
		t.Fatalf("src InitRepo() error: %v", err)
	}

	// Add the remote database as a file:// remote.
	remoteURL := fmt.Sprintf("file://%s/testdb", remoteDir)
	if err := srcSync.AddRemote("origin", remoteURL); err != nil {
		t.Fatalf("AddRemote() error: %v", err)
	}

	// Create a table and insert data.
	if _, err := srcDB.Exec("CREATE TABLE tasks (id VARCHAR(255) PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := srcDB.Exec("INSERT INTO tasks VALUES ('test-1')"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Push should snapshot + push without error.
	if err := srcSync.Push(); err != nil {
		t.Fatalf("Push() error: %v", err)
	}

	// Push again with no new changes should also succeed (no-op snapshot, push is idempotent).
	if err := srcSync.Push(); err != nil {
		t.Fatalf("second Push() error: %v", err)
	}
}

func TestPull_WithRemote(t *testing.T) {
	// Create a database, push to a file remote, then pull back.
	// Both use the same database instance so the tracking is consistent.
	remoteDir := t.TempDir()
	remoteDSN := fmt.Sprintf("file://%s?commitname=test&commitemail=test@test&database=testdb", remoteDir)
	remoteDB, err := sql.Open("dolt", remoteDSN)
	if err != nil {
		t.Fatalf("open remote dolt: %v", err)
	}
	for _, q := range []string{
		"DROP DATABASE IF EXISTS testdb",
		"CREATE DATABASE testdb",
		"USE testdb",
	} {
		if _, err := remoteDB.Exec(q); err != nil {
			t.Fatalf("init remote db (%s): %v", q, err)
		}
	}
	remoteSync := New(remoteDB)
	if err := remoteSync.InitRepo(); err != nil {
		t.Fatalf("remote InitRepo() error: %v", err)
	}
	remoteDB.Close()

	// Set up a local database, push to establish tracking, then pull.
	localDB := testDB(t)
	localSync := New(localDB)
	if err := localSync.InitRepo(); err != nil {
		t.Fatalf("local InitRepo() error: %v", err)
	}

	remoteURL := fmt.Sprintf("file://%s/testdb", remoteDir)
	if err := localSync.AddRemote("origin", remoteURL); err != nil {
		t.Fatalf("AddRemote() error: %v", err)
	}

	// Push --set-upstream to overwrite the remote and establish branch tracking.
	if _, err := localDB.Exec("CALL DOLT_PUSH('--force', '--set-upstream', 'origin', 'main')"); err != nil {
		t.Fatalf("force push: %v", err)
	}

	// Now pull should succeed since main tracks origin/main.
	if err := localSync.Pull(); err != nil {
		t.Fatalf("Pull() error: %v", err)
	}
}
