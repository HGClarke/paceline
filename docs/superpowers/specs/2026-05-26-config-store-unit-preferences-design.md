# Design: Config Store & Unit Preferences

**Date:** 2026-05-26  
**Status:** Approved  
**Scope:** v1 — `units` key only (`"metric"` | `"imperial"`). Future keys (`hr_max`, `ftp`, `anthropic_api_key`) are added in their respective feature PRs.

---

## Overview

All PacelineCLI output currently displays raw SI units — meters, m/s, meters for elevation. This design adds a persistent config file at `~/.paceline/config.toml` and a `paceline config` command family so users can switch to imperial display. Conversion happens at display time only; the database stores SI units unchanged.

---

## Architecture

### New files

```
internal/config/
  config.go        ← Config struct, Load(), Save(), DefaultPath()
  config_test.go
cmd/config.go      ← paceline config / config set / config get commands
```

### Modified files

```
cmd/root.go                   ← add PersistentPreRunE to load config into pkg-level cfg var
internal/display/table.go     ← Print* functions gain units string param; unit helpers added
internal/display/display_test.go ← tests updated to pass units
```

No changes to `store/`, `parser/`, `chart.go`, or `tui.go`.

### Data flow

```
startup
  → PersistentPreRunE loads ~/.paceline/config.toml → pkg-level cfg *config.Config
  → cmd/*.go RunE reads cfg.Units, passes to display.Print*()
  → display.Print*() calls formatDistance(v, units), formatSpeed(v, units), formatElevation(v, units)
```

---

## `internal/config` Package

```go
type Config struct {
    Units string `toml:"units"` // "metric" (default) | "imperial"
}

func DefaultPath() (string, error)  // → ~/.paceline/config.toml
func Load() (*Config, error)        // reads file; returns defaults if missing
func Save(cfg *Config) error        // writes file, creates ~/.paceline/ dir if needed
```

**Defaults:** `Load()` returns `&Config{Units: "metric"}` when the file does not exist — no error, no prompt.

**Forward compatibility:** Unknown TOML keys in the file are silently ignored so that a config written by a future version of the tool does not break an older binary.

**Validation:** `Load()` rejects any file where `units` is neither `"metric"` nor `"imperial"`, returning a descriptive error. Valid values are the only constraint enforced at load time.

**TOML library:** `github.com/BurntSushi/toml`.

---

## `cmd/config.go` — CLI Commands

### Commands

| Command | Description |
|---|---|
| `paceline config` | Print all config keys and current values as a table |
| `paceline config set <key> <value>` | Update one key; persist to file |
| `paceline config get <key>` | Print a single value (scriptable) |

### Validation on `config set`

- Unknown key → error listing valid keys: `unknown config key "foo"; valid keys: units`
- Invalid value for `units` → `units must be "metric" or "imperial"`

### Wiring into rootCmd

`rootCmd` gains a `PersistentPreRunE` that calls `config.Load()` and stores the result in a package-level `var cfg *config.Config`. All existing `RunE` bodies are unchanged — they simply pass `cfg.Units` to the display functions they already call.

---

## Display Changes (`internal/display/table.go`)

### Signature changes

Each `Print*` function gains one parameter:

```go
func PrintRideList(w io.Writer, rides []parser.Ride, total, page, limit int, jsonOut bool, units string)
func PrintRideDetail(w io.Writer, r parser.Ride, jsonOut bool, units string)
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool, units string)
```

### Unit-conversion helpers (private)

```go
func formatDistance(m float64, units string) string
    // metric:   "48.3 km"
    // imperial: "30.0 mi"

func formatSpeed(mps float64, units string) string
    // metric:   "28.1 km/h"
    // imperial: "17.5 mph"

func formatElevation(m float64, units string) string
    // metric:   "1,420 m"
    // imperial: "4,659 ft"
```

### Conversion constants

| Measurement | Factor |
|---|---|
| meters → miles | ÷ 1609.344 |
| m/s → mph | × 2.23694 |
| meters → feet | × 3.28084 |

`PrintStreamChart` (chart.go) Y-axis labels are left unchanged in v1; that can be updated in a follow-up.

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| Config file missing | `Load()` returns defaults — not an error |
| Config file malformed TOML | `PersistentPreRunE` surfaces error: `config: <toml parse detail>` |
| `units` value invalid in file | `Load()` returns error: `units must be "metric" or "imperial"` |
| `config set` unknown key | Error: `unknown config key "foo"; valid keys: units` |
| `config set units` bad value | Error: `units must be "metric" or "imperial"` |
| Save fails (permissions) | `config set` returns OS-level error |

---

## Testing

### `internal/config/config_test.go`

- Round-trip: `Save` then `Load` returns the same values
- Missing file: `Load` returns `&Config{Units: "metric"}`, no error
- Malformed TOML: `Load` returns error
- Invalid `units` value: `Load` returns error
- `DefaultPath` returns a path ending in `config.toml`

### `internal/display/display_test.go`

- Existing tests updated: pass `"metric"` to `Print*` functions (behaviour unchanged)
- New cases: assert imperial formatting for distance (mi), speed (mph), elevation (ft)

### Manual verification

```bash
paceline config                        # shows: units  metric
paceline config set units imperial
paceline config                        # shows: units  imperial
paceline rides                         # distances in miles, speeds in mph
paceline ride 1                        # elevation in feet
paceline stats                         # all values imperial
paceline config set units metric       # revert
paceline config set units bad          # error message
paceline config set foo bar            # unknown key error
```
