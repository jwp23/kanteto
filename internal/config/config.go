package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds application configuration.
type Config struct {
	ActiveProfile string `toml:"-"`
}

// tomlConfig is the on-disk representation.
type tomlConfig struct {
	ActiveProfile string `toml:"active_profile"`
}

// Load reads config from XDG_CONFIG_HOME/kanteto/config.toml.
// Returns defaults if the file does not exist.
func Load() (Config, error) {
	cfg := Config{
		ActiveProfile: "default",
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

	if tc.ActiveProfile != "" {
		cfg.ActiveProfile = tc.ActiveProfile
	}

	return cfg, nil
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
