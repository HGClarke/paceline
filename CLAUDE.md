# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Install linter (one-time, adds to ~/go/bin/)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build
make build          # go build -o paceline .
go build -o paceline .   # also works directly

# Run without building
go run . <command>

# Run all checks (vet + tests + lint)
make all

# Run tests
make test           # go test ./...
go test ./...       # also works directly

# Run tests for a single package
go test ./internal/parser/...
go test ./internal/store/...
go test ./internal/display/...
go test ./internal/config/...

# Run a single test by name
go test ./internal/parser/... -run TestParseFIT

# Vet
make vet            # go vet ./...
go vet ./...        # also works directly

# Lint
make lint           # golangci-lint run ./...
```

The compiled binary writes its database to `~/.paceline/data.db` (DuckDB). Test data files are in `testdata/` (sample.fit, sample.gpx, sample.tcx).

## Architecture

```
main.go              → calls cmd.Execute()
cmd/                 → Cobra commands (one file per command)
internal/
  parser/            → file format parsers → Ride + []Stream structs
  store/             → DuckDB persistence (rides + streams tables)
  display/           → rendering: tables, ASCII charts, bubbletea TUI
  config/            → user config (TOML): Load, Save, DefaultPath
```

### Data flow

1. **`parser`** — `ParseGPX`, `ParseTCX`, `ParseFIT` each return `(*parser.Ride, []parser.Stream, error)`. All optional sensor fields (HR, power, cadence) are `*int`/`*float64` (nil when absent).
2. **`store`** — wraps a `*sql.DB` (DuckDB). `migrate()` runs DDL on open; import is idempotent (`ON CONFLICT (filename) DO NOTHING`). `Store.DefaultPath()` returns `~/.paceline/data.db`.
3. **`display`** — three rendering paths selected at call site:
   - `PrintRideList` / `PrintRideDetail` / `PrintStats` / `PrintRecords` → `tablewriter` tables (or JSON when `--json` is set)
   - `PrintStreamChart` → `asciigraph` ASCII line charts
   - `RunRidesTUI` → interactive `bubbletea` TUI (only when `display.IsTTY()` is true)

### Cobra routing workaround

`ride <id> stream [--field=...]` has a non-standard dispatch. Because `<id>` is numeric, Cobra cannot route through it to find the `stream` subcommand automatically. `rideCmd.RunE` (`cmd/ride.go`) manually inspects `args[1]` and delegates to matching subcommands via `sub.RunE`. The `--field` flag is also mirrored as a hidden flag on `rideCmd` (kept in sync with `streamCmd`) so Cobra parses it in the manual routing path. When adding new subcommands to `rideCmd`, this manual dispatch loop picks them up automatically.

### Stream fields

Valid field names for `ride <id> stream --field=<name>`: `power`, `hr`, `speed`, `cadence`, `altitude`. The mapping to DB columns lives in `store.fieldColumn`. Default auto-selection priority: power → hr → speed.

### `--json` flag

Defined as a persistent flag on `rootCmd`; the `jsonOutput` bool is package-level in `cmd/` and passed down to display functions. All `Print*` functions accept a `jsonOut bool` parameter.
