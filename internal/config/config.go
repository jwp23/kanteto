package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultLeadTime is the default reminder lead time before a task is due.
const DefaultLeadTime = 15 * time.Minute

// Config holds application configuration.
type Config struct {
	ReminderLeadTime time.Duration `toml:"-"`
	ActiveProfile    string        `toml:"-"`
}

// tomlConfig is the on-disk representation with string durations.
type tomlConfig struct {
	ReminderLeadTime string `toml:"reminder_lead_time"`
	ActiveProfile    string `toml:"active_profile"`
}

// Load reads config from XDG_CONFIG_HOME/kanteto/config.toml.
// Returns defaults if the file does not exist.
func Load() (Config, error) {
	cfg := Config{
		ReminderLeadTime: DefaultLeadTime,
		ActiveProfile:    "default",
	}

	path := filepath.Join(configDir(), "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var tc tomlConfig
	if err := toml.Unmarshal(data, &tc); err != nil {
		return cfg, err
	}

	if tc.ReminderLeadTime != "" {
		d, err := time.ParseDuration(tc.ReminderLeadTime)
		if err != nil {
			return cfg, err
		}
		cfg.ReminderLeadTime = d
	}
	if tc.ActiveProfile != "" {
		cfg.ActiveProfile = tc.ActiveProfile
	}

	return cfg, nil
}

// Save writes the config to XDG_CONFIG_HOME/kanteto/config.toml.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tc := tomlConfig{
		ActiveProfile: cfg.ActiveProfile,
	}
	if cfg.ReminderLeadTime != DefaultLeadTime {
		tc.ReminderLeadTime = cfg.ReminderLeadTime.String()
	}

	path := filepath.Join(dir, "config.toml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(tc)
}

// DataDir returns the XDG-compliant data directory for kanteto.
func DataDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "kanteto")
}

func configDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "kanteto")
}
