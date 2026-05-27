# PacelineCLI Design Spec
**Date:** 2026-05-23

## Overview

PacelineCLI is a Go CLI tool that ingests ride data files (.fit, .gpx, .tcx), stores structured data in a local DuckDB database, and surfaces stats, aggregates, and time-series charts via terminal commands.

---

## Architecture

A single Go binary (`paceline`) built with a Cobra command tree. On first run it initializes a DuckDB database at `~/.paceline/data.db`. All commands read from and write to this database — original ride files are only touched at import time.

Two internal layers:
- **Parsers** — one per format (.fit, .gpx, .tcx), each producing a shared `Ride` and `[]Stream` struct
- **Store** — DuckDB-backed repository that reads/writes rides and streams

The CLI (`cmd/`) layer wires parsers and store together. Neither layer depends on the other.

---

## Data Model

### `rides` table
One row per imported ride.

| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | auto-increment |
| filename | TEXT | used for idempotent re-import |
| recorded_at | TIMESTAMP | start time of the ride |
| distance_m | DOUBLE | total distance in metres |
| duration_s | INTEGER | total elapsed seconds |
| elevation_gain_m | DOUBLE | cumulative ascent in metres |
| avg_speed_mps | DOUBLE | |
| max_speed_mps | DOUBLE | |
| avg_hr_bpm | INTEGER | null if not recorded |
| max_hr_bpm | INTEGER | null if not recorded |
| avg_power_w | INTEGER | null if not recorded |
| max_power_w | INTEGER | null if not recorded |
| calories | INTEGER | null if not recorded |
| source_format | TEXT | "fit", "gpx", or "tcx" |

### `streams` table
One row per data point within a ride (typically one per second or per GPS fix).

| Column | Type | Notes |
|---|---|---|
| ride_id | INTEGER FK | references rides.id |
| timestamp | TIMESTAMP | absolute time of the data point |
| elapsed_s | INTEGER | seconds since ride start |
| speed_mps | DOUBLE | null if not recorded |
| hr_bpm | INTEGER | null if not recorded |
| power_w | INTEGER | null if not recorded |
| cadence_rpm | INTEGER | null if not recorded |
| altitude_m | DOUBLE | null if not recorded |
| lat | DOUBLE | null if not recorded |
| lon | DOUBLE | null if not recorded |

A 1-hour ride at 1-second resolution produces ~3,600 rows in `streams`, all sharing the same `ride_id` with unique `timestamp`/`elapsed_s` values. DuckDB's columnar storage handles window functions and time-range aggregates over these rows efficiently.

---

## Command Surface

```
paceline import <file|directory>
  Parse and store ride(s) into DuckDB.
  Idempotent by filename — re-importing the same file is a no-op.
  Directory imports skip malformed files and report a summary.

paceline rides [--year=YYYY] [--month=MM] [--date=YYYY-MM-DD]
               [--page=N] [--limit=N]
  List rides. Defaults to the 10 most recent if no filters are given.
  Filters are all optional and composable. --month without --year defaults
  to the current calendar year. --limit defaults to 10 in all cases.
  Auto-detects TTY: interactive paginated TUI in a terminal,
  plain table output when piped.

paceline ride <id>
  Summary stats for a specific ride (all columns from the rides table,
  formatted as a human-readable table).

paceline ride <id> stream [--field=power|hr|speed|cadence|altitude]
  Terminal line chart of a stream field for a specific ride.
  Defaults to power if available, then hr, then speed.
  If the requested field has no data, lists which fields are available.

paceline stats [--year=YYYY] [--month=MM] [--week=N]
  Aggregated totals (distance, duration, elevation, ride count).
  Filters are optional and composable.
  --week is the ISO week number (1–53) within the given year.
  --month without --year defaults to the current calendar year.
  --week without --year defaults to the current calendar year.
  Defaults to the current calendar month if no flags are given.
```

Global flags available on all commands:
- `--json` — emit JSON instead of tables (ignored in TUI mode)

---

## Interactive TUI (rides list)

When `paceline rides` detects a real TTY, it launches a `bubbletea` interactive list:
- `↑`/`↓` to move between rows
- `n`/`p` or `→`/`←` for next/previous page
- `enter` to drill into a ride (shows the same output as `paceline ride <id>`)
- `q` or `esc` to quit

When stdout is a pipe, falls back to plain `tablewriter` output with a footer:
```
Page 1 of 7  —  run with --page=2 for next
```

---

## Project Structure

```
paceline/
├── cmd/
│   ├── root.go          # cobra root, global flags
│   ├── import.go        # paceline import
│   ├── rides.go         # paceline rides
│   ├── ride.go          # paceline ride <id>
│   ├── stream.go        # paceline ride <id> stream
│   └── stats.go         # paceline stats
├── internal/
│   ├── parser/
│   │   ├── fit.go       # .fit parser → Ride + []Stream
│   │   ├── gpx.go       # .gpx parser → Ride + []Stream
│   │   ├── tcx.go       # .tcx parser → Ride + []Stream
│   │   └── model.go     # shared Ride and Stream structs
│   ├── store/
│   │   ├── db.go        # DuckDB connection, init, migrations
│   │   ├── rides.go     # ride CRUD and filter queries
│   │   └── streams.go   # stream insert and time-range queries
│   └── display/
│       ├── table.go     # tablewriter plain output
│       ├── chart.go     # asciigraph terminal line charts
│       └── tui.go       # bubbletea interactive list + pagination
└── main.go
```

---

## Dependency Stack

| Purpose | Library |
|---|---|
| Command structure | `github.com/spf13/cobra` |
| DuckDB | `github.com/marcboeker/go-duckdb` |
| .fit parsing | `github.com/tormoder/fit` |
| .gpx parsing | `github.com/tkrajina/gpxgo` |
| .tcx parsing | `encoding/xml` (stdlib) |
| Terminal tables | `github.com/olekukonko/tablewriter` |
| Terminal charts | `github.com/guptarohit/asciigraph` |
| Interactive TUI | `github.com/charmbracelet/bubbletea` |

**DuckDB licensing:** MIT-licensed, fully free and open-source. No tiers, no data limits, no cloud dependency. Runs entirely locally. Requires CGO at build time.

---

## Error Handling

- **Malformed import file:** skip the file, print a warning, continue. Report summary at end.
- **Directory import:** `3 imported, 1 skipped: bad.fit — unrecognized format`
- **Missing ride ID:** friendly message + non-zero exit code
- **DB not initialized:** created automatically on first run, no manual setup needed
- **Missing stream field:** if `--field=power` has no data for a ride, print which fields are available and exit cleanly

---

## Testing

- Unit tests for each parser against small sample files checked into the repo under `testdata/`
- Unit tests for store queries using an in-memory DuckDB instance (`:memory:`)
- No full CLI integration tests in v1
