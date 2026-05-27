# Config Store & Unit Preferences Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a persistent `~/.paceline/config.toml` file that controls metric vs imperial display, a `paceline config` command family for managing it, and unit-aware formatting across all display functions.

**Architecture:** A new `internal/config` package owns the Config struct and TOML file I/O. `cmd/root.go` loads config into a package-level `cfg` var via `loadCfg()`, which each command passes to `display.Print*()`. Display functions gain a `units string` parameter and delegate to private format helpers for distance, speed, and elevation.

**Tech Stack:** `github.com/BurntSushi/toml v1.2.1` (already in go.mod as indirect dep), Cobra, existing tablewriter pattern.

---

## File Map

| Action | File | Responsibility |
|---|---|---|
| Create | `internal/config/config.go` | Config struct, Load, LoadFrom, Save, SaveTo, DefaultPath |
| Create | `internal/config/config_test.go` | Unit tests for config package |
| Create | `cmd/config.go` | `paceline config`, `config set`, `config get` commands |
| Modify | `cmd/root.go` | Add `cfg *config.Config`, `loadCfg()`, `PersistentPreRunE` |
| Modify | `cmd/ride.go` | Chain `loadCfg()` in existing `PersistentPreRunE`; pass `cfg.Units` to `PrintRideDetail` |
| Modify | `cmd/rides.go` | Pass `cfg.Units` to `PrintRideList` and `PrintRideDetail` |
| Modify | `cmd/stats.go` | Pass `cfg.Units` to `PrintStats` |
| Modify | `internal/display/table.go` | Add format helpers; update `Print*` signatures |
| Modify | `internal/display/display_test.go` | Update signatures; add imperial assertions |

---

## Task 1: `internal/config` Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1.1: Write failing tests**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hollandclarke/paceline/internal/config"
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
```

- [ ] **Step 1.2: Run tests to confirm they fail**

```bash
go test ./internal/config/...
```

Expected: compile error — package does not exist yet.

- [ ] **Step 1.3: Implement `internal/config/config.go`**

```go
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
```

- [ ] **Step 1.4: Run tests to confirm they pass**

```bash
go test ./internal/config/...
```

Expected: all 8 tests PASS.

- [ ] **Step 1.5: Run go mod tidy to make toml a direct dependency**

```bash
go mod tidy
```

Expected: `go.mod` moves `github.com/BurntSushi/toml` from `// indirect` to a direct require.

- [ ] **Step 1.6: Vet and commit**

```bash
go vet ./internal/config/...
git add internal/config/ go.mod go.sum
git commit -m "feat: add internal/config package with TOML load/save"
```

---

## Task 2: Wire Config Loading into `cmd` Layer

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/ride.go`

**Background:** Cobra calls `PersistentPreRunE` from the nearest ancestor that defines it. Since `rideCmd` already defines its own `PersistentPreRunE`, it shadows `rootCmd.PersistentPreRunE` for the `ride` family. The fix is a shared `loadCfg()` helper called from both.

- [ ] **Step 2.1: Update `cmd/root.go`**

Replace the entire file content:

```go
package cmd

import (
	"fmt"

	"github.com/hollandclarke/paceline/internal/config"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "paceline",
	Short: "CLI for analyzing cycling ride data",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadCfg()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

// loadCfg loads the user config file into the package-level cfg variable.
// It is called from rootCmd.PersistentPreRunE and from any subcommand that
// defines its own PersistentPreRunE (which would otherwise shadow the root one).
func loadCfg() error {
	c, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	cfg = c
	return nil
}
```

- [ ] **Step 2.2: Update `cmd/ride.go` — chain `loadCfg()` in `rideCmd.PersistentPreRunE`**

In `cmd/ride.go`, replace the `PersistentPreRunE` field in the `rideCmd` declaration. The full updated `rideCmd` declaration (lines 23–53 in the original):

```go
var rideCmd = &cobra.Command{
	Use:   "ride <position>",
	Short: "Show summary stats for a specific ride",
	Args:  cobra.ArbitraryArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Cobra does not chain PersistentPreRunE from parent when the child
		// defines its own, so we call the root loader explicitly.
		if err := loadCfg(); err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("requires a ride position")
		}
		pos, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid position %q: must be a number", args[0])
		}
		dbPath, err := store.DefaultPath()
		if err != nil {
			return err
		}
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer s.Close()
		ride, err := s.GetRideByPosition(pos)
		if err != nil {
			return err
		}
		currentRide = ride
		currentRideID = ride.ID
		return nil
	},
	RunE: runRide,
}
```

- [ ] **Step 2.3: Verify the build compiles cleanly**

```bash
go build ./...
```

Expected: no errors. (Display functions still have old signatures — call sites not yet updated.)

- [ ] **Step 2.4: Commit**

```bash
git add cmd/root.go cmd/ride.go
git commit -m "feat: load config in PersistentPreRunE; expose loadCfg helper"
```

---

## Task 3: Unit Format Helpers in `display`

**Files:**
- Modify: `internal/display/table.go` (add helpers only — no signature changes yet)
- Modify: `internal/display/display_test.go` (add helper tests)

- [ ] **Step 3.1: Write failing tests for the format helpers**

Append to `internal/display/display_test.go` (after the existing `TestPrintRideList_ShowsPositionColumn` test):

```go
func TestFormatDistance_Metric(t *testing.T) {
	tests := []struct {
		m    float64
		want string
	}{
		{1000, "1.0 km"},
		{48280, "48.3 km"},
		{0, "0.0 km"},
	}
	for _, tt := range tests {
		got := formatDistance(tt.m, "metric")
		if got != tt.want {
			t.Errorf("formatDistance(%.0f, metric) = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFormatDistance_Imperial(t *testing.T) {
	tests := []struct {
		m    float64
		want string
	}{
		{1609.344, "1.0 mi"},
		{80467.2, "50.0 mi"},
	}
	for _, tt := range tests {
		got := formatDistance(tt.m, "imperial")
		if got != tt.want {
			t.Errorf("formatDistance(%.3f, imperial) = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFormatSpeed_Metric(t *testing.T) {
	// 10 m/s = 36 km/h
	got := formatSpeed(10.0, "metric")
	if got != "36.0 km/h" {
		t.Errorf("formatSpeed(10, metric) = %q, want %q", got, "36.0 km/h")
	}
}

func TestFormatSpeed_Imperial(t *testing.T) {
	// 1 m/s = 2.23694 mph → round to 1dp = "2.2 mph"
	got := formatSpeed(1.0, "imperial")
	if got != "2.2 mph" {
		t.Errorf("formatSpeed(1, imperial) = %q, want %q", got, "2.2 mph")
	}
}

func TestFormatElevation_Metric(t *testing.T) {
	got := formatElevation(1420.0, "metric")
	if got != "1420 m" {
		t.Errorf("formatElevation(1420, metric) = %q, want %q", got, "1420 m")
	}
}

func TestFormatElevation_Imperial(t *testing.T) {
	// 1 m = 3.28084 ft; 100 m = 328 ft
	got := formatElevation(100.0, "imperial")
	if got != "328 ft" {
		t.Errorf("formatElevation(100, imperial) = %q, want %q", got, "328 ft")
	}
}
```

- [ ] **Step 3.2: Run tests to confirm they fail**

```bash
go test ./internal/display/...
```

Expected: compile error — `formatDistance`, `formatSpeed`, `formatElevation` undefined.

- [ ] **Step 3.3: Add the helpers to `internal/display/table.go`**

Append these private functions at the bottom of `internal/display/table.go`, after `formatDuration`:

```go
// formatDistance formats meters as "X.X km" (metric) or "X.X mi" (imperial).
func formatDistance(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1f mi", m/1609.344)
	}
	return fmt.Sprintf("%.1f km", m/1000)
}

// formatSpeed formats m/s as "X.X km/h" (metric) or "X.X mph" (imperial).
func formatSpeed(mps float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1f mph", mps*2.23694)
	}
	return fmt.Sprintf("%.1f km/h", mps*3.6)
}

// formatElevation formats meters as "X m" (metric) or "X ft" (imperial).
func formatElevation(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.0f ft", m*3.28084)
	}
	return fmt.Sprintf("%.0f m", m)
}
```

- [ ] **Step 3.4: Run tests to confirm helpers pass**

```bash
go test ./internal/display/...
```

Expected: all tests PASS (existing tests unaffected; new helper tests pass).

- [ ] **Step 3.5: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go
git commit -m "feat: add unit format helpers to display package"
```

---

## Task 4: Update `Print*` Signatures and All Call Sites

**Files:**
- Modify: `internal/display/table.go` — update three `Print*` signatures to accept `units string`
- Modify: `internal/display/display_test.go` — update existing test + add imperial table test
- Modify: `cmd/rides.go` — pass `cfg.Units`
- Modify: `cmd/stats.go` — pass `cfg.Units`
- Modify: `cmd/ride.go` — pass `cfg.Units`

- [ ] **Step 4.1: Update `display_test.go` first (tests will fail until implementations follow)**

In `internal/display/display_test.go`, update `TestPrintRideList_ShowsPositionColumn` and add an imperial test. Replace the existing `TestPrintRideList_ShowsPositionColumn` function and append the new test:

```go
func TestPrintRideList_ShowsPositionColumn(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:       42,
			RecordedAt:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:      30000,
			DurationS:      3600,
			ElevationGainM: 500,
			AvgSpeedMPS:    8.3,
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, false, "metric")
	output := buf.String()
	if !strings.Contains(output, "#") {
		t.Errorf("expected '#' column header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("expected position '42' in output, got:\n%s", output)
	}
}

func TestPrintRideList_Imperial(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:       1,
			RecordedAt:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:      16093.44, // exactly 10 miles
			DurationS:      3600,
			ElevationGainM: 304.8, // exactly 1000 ft
			AvgSpeedMPS:    4.4704, // exactly 10 mph
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, false, "imperial")
	output := buf.String()
	if !strings.Contains(output, "mi") {
		t.Errorf("expected miles unit in output, got:\n%s", output)
	}
	if !strings.Contains(output, "mph") {
		t.Errorf("expected mph unit in output, got:\n%s", output)
	}
	if !strings.Contains(output, "ft") {
		t.Errorf("expected ft unit in output, got:\n%s", output)
	}
	if strings.Contains(output, " km") {
		t.Errorf("expected no km in imperial output, got:\n%s", output)
	}
}
```

- [ ] **Step 4.2: Run tests — confirm compile error on wrong Print* arity**

```bash
go test ./internal/display/...
```

Expected: compile error — `PrintRideList` called with wrong number of arguments.

- [ ] **Step 4.3: Update `Print*` signatures in `internal/display/table.go`**

Replace the three function signatures and their bodies to use the helpers. Here is the full updated content of the three functions (leave `formatDuration` and the helpers from Task 3 unchanged):

```go
// PrintRideList renders a table of rides to w. If jsonOut is true, emits JSON instead.
func PrintRideList(w io.Writer, rides []parser.Ride, total, page, limit int, jsonOut bool, units string) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(rides)
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(tablewriter.WithBorders(tw.Border{
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"#", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
	for _, r := range rides {
		table.Append([]string{
			strconv.FormatInt(r.Position, 10),
			r.RecordedAt.Format("2006-01-02"),
			formatDistance(r.DistanceM, units),
			formatDuration(r.DurationS),
			formatElevation(r.ElevationGainM, units),
			formatSpeed(r.AvgSpeedMPS, units),
		})
	}
	table.Render()
	if limit > 0 {
		pages := int(math.Ceil(float64(total) / float64(limit)))
		if pages > 1 {
			fmt.Fprintf(w, "\nPage %d of %d  —  run with --page=%d for next\n", page, pages, page+1)
		}
	}
}

// PrintRideDetail renders a single ride's full summary to w.
func PrintRideDetail(w io.Writer, r parser.Ride, jsonOut bool, units string) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(r)
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	rows := [][]string{
		{"Date", r.RecordedAt.Format("2006-01-02")},
		{"Distance", formatDistance(r.DistanceM, units)},
		{"Duration", formatDuration(r.DurationS)},
		{"Elevation Gain", formatElevation(r.ElevationGainM, units)},
		{"Avg Speed", formatSpeed(r.AvgSpeedMPS, units)},
		{"Max Speed", formatSpeed(r.MaxSpeedMPS, units)},
		{"Format", r.SourceFormat},
	}
	if r.AvgHRBPM != nil {
		rows = append(rows, []string{"Avg HR", fmt.Sprintf("%d bpm", *r.AvgHRBPM)})
	}
	if r.MaxHRBPM != nil {
		rows = append(rows, []string{"Max HR", fmt.Sprintf("%d bpm", *r.MaxHRBPM)})
	}
	if r.AvgPowerW != nil {
		rows = append(rows, []string{"Avg Power", fmt.Sprintf("%d W", *r.AvgPowerW)})
	}
	if r.MaxPowerW != nil {
		rows = append(rows, []string{"Max Power", fmt.Sprintf("%d W", *r.MaxPowerW)})
	}
	if r.Calories != nil {
		rows = append(rows, []string{"Calories", strconv.Itoa(*r.Calories)})
	}
	table.Bulk(rows)
	table.Render()
}

// PrintStats renders aggregated stats to w.
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool, units string) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(st)
		return
	}
	fmt.Fprintf(w, "Stats: %s\n\n", label)
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Bulk([][]string{
		{"Rides", strconv.Itoa(st.RideCount)},
		{"Total Distance", formatDistance(st.TotalDistanceM, units)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", formatElevation(st.TotalElevationM, units)},
	})
	table.Render()
}
```

- [ ] **Step 4.4: Run display tests — confirm they now pass**

```bash
go test ./internal/display/...
```

Expected: all tests PASS.

- [ ] **Step 4.5: Update call sites in `cmd/rides.go`**

In `cmd/rides.go`, update both `PrintRideDetail` and `PrintRideList` calls:

Line 85 — change:
```go
display.PrintRideDetail(os.Stdout, *selected, false)
```
to:
```go
display.PrintRideDetail(os.Stdout, *selected, false, cfg.Units)
```

Line 90 — change:
```go
display.PrintRideList(os.Stdout, rides, total, ridesPage, ridesLimit, jsonOutput)
```
to:
```go
display.PrintRideList(os.Stdout, rides, total, ridesPage, ridesLimit, jsonOutput, cfg.Units)
```

- [ ] **Step 4.6: Update call site in `cmd/stats.go`**

Line 100 — change:
```go
display.PrintStats(os.Stdout, st, label, jsonOutput)
```
to:
```go
display.PrintStats(os.Stdout, st, label, jsonOutput, cfg.Units)
```

- [ ] **Step 4.7: Update call site in `cmd/ride.go`**

Line 88 — change:
```go
display.PrintRideDetail(os.Stdout, currentRide, jsonOutput)
```
to:
```go
display.PrintRideDetail(os.Stdout, currentRide, jsonOutput, cfg.Units)
```

- [ ] **Step 4.8: Build and test everything**

```bash
go build ./...
go test ./...
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 4.9: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go \
        cmd/rides.go cmd/stats.go cmd/ride.go
git commit -m "feat: add units param to Print* functions; wire cfg.Units at call sites"
```

---

## Task 5: `cmd/config.go` Command Family

**Files:**
- Create: `cmd/config.go`

- [ ] **Step 5.1: Implement `cmd/config.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/hollandclarke/paceline/internal/config"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	RunE:  runConfig,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Options(tablewriter.WithBorders(tw.Border{
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"Key", "Value"})
	table.Append([]string{"units", cfg.Units})
	table.Render()
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	switch key {
	case "units":
		if value != "metric" && value != "imperial" {
			return fmt.Errorf(`units must be "metric" or "imperial"`)
		}
		cfg.Units = value
	default:
		return fmt.Errorf("unknown config key %q; valid keys: units", key)
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	switch key {
	case "units":
		fmt.Fprintln(os.Stdout, cfg.Units)
	default:
		return fmt.Errorf("unknown config key %q; valid keys: units", key)
	}
	return nil
}
```

- [ ] **Step 5.2: Build and run all tests**

```bash
go build ./...
go test ./...
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 5.3: Manual smoke test**

```bash
go run . config
# KEY    VALUE
# units  metric

go run . config set units imperial
# Set units = imperial

go run . config
# KEY    VALUE
# units  imperial

go run . config get units
# imperial

go run . config set units metric
# Set units = metric

go run . config set units bad
# Error: units must be "metric" or "imperial"

go run . config set foo bar
# Error: unknown config key "foo"; valid keys: units
```

- [ ] **Step 5.4: Commit**

```bash
git add cmd/config.go
git commit -m "feat: add paceline config / config set / config get commands"
```

---

## Task 6: End-to-End Verification

- [ ] **Step 6.1: Full test suite**

```bash
go test ./...
go vet ./...
```

Expected: all tests PASS, no vet warnings.

- [ ] **Step 6.2: Manual end-to-end with real data**

```bash
# Switch to imperial and verify all commands respect it
go run . config set units imperial
go run . rides          # distances in mi, speeds in mph, elevation in ft
go run . ride 1         # detail view: mi, mph, ft
go run . stats          # total distance in mi, elevation in ft

# Switch back to metric
go run . config set units metric
go run . rides          # distances back in km
```

- [ ] **Step 6.3: Verify JSON output is unaffected**

```bash
go run . rides --json | head -5
# Should show raw SI values (DistanceM in meters) — JSON always bypasses formatting
```

- [ ] **Step 6.4: Final commit if any fixes were made, otherwise done**

```bash
git log --oneline -6
# Should show the 5 feature commits from Tasks 1–5
```
