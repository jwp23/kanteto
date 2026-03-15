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
	if cfg.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want %q", cfg.ActiveProfile, "default")
	}
	if cfg.SoundFile != "" {
		t.Errorf("SoundFile = %q, want empty", cfg.SoundFile)
	}
}

func TestLoad_FromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kanteto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `active_profile = "work"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want work", cfg.ActiveProfile)
	}
}

func TestLoad_SoundFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kanteto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `active_profile = "default"
sound_file = "/usr/share/sounds/freedesktop/stereo/complete.oga"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.SoundFile != "/usr/share/sounds/freedesktop/stereo/complete.oga" {
		t.Errorf("SoundFile = %q, want sound file path", cfg.SoundFile)
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
