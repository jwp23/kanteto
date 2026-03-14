package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Use a temp dir so no real config is found
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ReminderLeadTime != DefaultLeadTime {
		t.Errorf("ReminderLeadTime = %v, want %v", cfg.ReminderLeadTime, DefaultLeadTime)
	}
}

func TestLoad_FromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kanteto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `reminder_lead_time = "30m"
active_profile = "work"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ReminderLeadTime.String() != "30m0s" {
		t.Errorf("ReminderLeadTime = %v, want 30m", cfg.ReminderLeadTime)
	}
	if cfg.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want work", cfg.ActiveProfile)
	}
}

func TestDataDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	dir := DataDir()
	expected := filepath.Join(tmp, "kanteto")
	if dir != expected {
		t.Errorf("DataDir() = %q, want %q", dir, expected)
	}
}
