package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds persistent user preferences. All fields have safe defaults.
type Config struct {
	Units string `toml:"units"` // "metric" (default) | "imperial"
}

// DefaultPath returns ~/.paceline/config.toml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".paceline", "config.toml"), nil
}

// Load reads config from DefaultPath. Returns defaults if the file does not
// exist. Returns an error if the file exists but cannot be parsed or contains
// invalid values.
func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads config from an explicit path. Exported for testing.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{Units: "metric"}
	_, err := toml.DecodeFile(path, cfg)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Units != "metric" && cfg.Units != "imperial" {
		return nil, fmt.Errorf(`units must be "metric" or "imperial", got %q`, cfg.Units)
	}
	return cfg, nil
}

// Save writes cfg to DefaultPath, creating ~/.paceline/ if needed.
func Save(cfg *Config) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return SaveTo(path, cfg)
}

// SaveTo writes cfg to the given path, creating parent directories if needed.
// Exported for testing.
func SaveTo(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
