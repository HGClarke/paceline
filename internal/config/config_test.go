package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HGClarke/paceline/internal/config"
)

func TestDefaultPath_EndsInConfigTOML(t *testing.T) {
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "config.toml") {
		t.Errorf("DefaultPath() = %q, want path ending in config.toml", path)
	}
}

func TestLoadFrom_MissingFile_ReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Units != "metric" {
		t.Errorf("Units = %q, want %q", cfg.Units, "metric")
	}
}

func TestLoadFrom_ValidMetric(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`units = "metric"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Units != "metric" {
		t.Errorf("Units = %q, want %q", cfg.Units, "metric")
	}
}

func TestLoadFrom_ValidImperial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`units = "imperial"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Units != "imperial" {
		t.Errorf("Units = %q, want %q", cfg.Units, "imperial")
	}
}

func TestLoadFrom_MalformedTOML_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("units = [not valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for malformed TOML, got nil")
	}
}

func TestLoadFrom_InvalidUnits_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`units = "furlongs"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid units value, got nil")
	}
	if !strings.Contains(err.Error(), "metric") || !strings.Contains(err.Error(), "imperial") {
		t.Errorf("error message should mention valid values, got: %v", err)
	}
}

func TestSaveTo_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := &config.Config{Units: "imperial"}
	if err := config.SaveTo(path, want); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}
	got, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if got.Units != want.Units {
		t.Errorf("round-trip Units = %q, want %q", got.Units, want.Units)
	}
}

func TestSaveTo_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "config.toml")
	cfg := &config.Config{Units: "metric"}
	if err := config.SaveTo(path, cfg); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

// Load and Save delegate to DefaultPath(), which reads $HOME. Redirecting $HOME
// to a temp dir lets us test them without touching the real ~/.paceline/config.toml.

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Units != "metric" {
		t.Errorf("Units = %q, want %q", cfg.Units, "metric")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".paceline"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".paceline", "config.toml")
	if err := os.WriteFile(path, []byte(`units = "imperial"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Units != "imperial" {
		t.Errorf("Units = %q, want %q", cfg.Units, "imperial")
	}
}

func TestLoad_InvalidFile_ReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".paceline"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".paceline", "config.toml")
	if err := os.WriteFile(path, []byte(`units = "furlongs"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid units, got nil")
	}
}

func TestSave_WritesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := &config.Config{Units: "imperial"}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	path := filepath.Join(home, ".paceline", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s, got: %v", path, err)
	}
}

func TestSave_And_Load_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	want := &config.Config{Units: "imperial"}
	if err := config.Save(want); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if got.Units != want.Units {
		t.Errorf("round-trip Units = %q, want %q", got.Units, want.Units)
	}
}
