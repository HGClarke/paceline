# 🚴 Paceline

**A fast, offline CLI for analyzing cycling ride data.**

Import `.fit`, `.gpx`, or `.tcx` files, then browse your rides, inspect stats, and chart sensor streams — all from the terminal. Data lives in a local [DuckDB](https://duckdb.org/) database; no accounts, no cloud, no subscriptions.

---

## Demo

```
$ paceline import ~/rides/
Importing 3 files...
  imported: 2026-05-20_morning.fit (id=1, 3602 points)
  imported: 2026-05-18_endurance.fit (id=2, 7514 points)
  imported: 2026-05-15_intervals.fit (id=3, 4230 points)
Done: 3 imported, 0 already exist, 0 errors
```

```
$ paceline rides
 #    DATE         DISTANCE    DURATION     ELEVATION   AVG SPEED
---  ----------   ----------  -----------  ----------  ----------
 1   2026-05-20   42.3 km     1h 22m 14s   580 m       30.8 km/h
 2   2026-05-18   68.1 km     2h 05m 42s   1240 m      32.6 km/h
 3   2026-05-15   38.7 km     1h 08m 30s   310 m       33.9 km/h
```

```
$ paceline ride 1
 DATE            2026-05-20
 DISTANCE        42.3 km
 DURATION        1h 22m 14s
 ELEVATION GAIN  580 m
 AVG SPEED       30.8 km/h
 MAX SPEED       52.1 km/h
 FORMAT          fit
 AVG HR          142 bpm
 MAX HR          178 bpm
 AVG POWER       195 W
 MAX POWER       412 W
 CALORIES        1240
```

```
$ paceline ride 1 stream --field=power
 412 ┤                                      ╭╮
 390 ┤                          ╭──╮   ╭────╯╰────╮
 368 ┤                   ╭──────╯  ╰───╯          ╰──╮
 347 ┤              ╭────╯                            ╰────╮
 325 ┤        ╭─────╯                                      ╰──
 303 ┤   ╭────╯
 281 ┤───╯
                               power
```

In a terminal, `paceline rides` launches an **interactive TUI** — navigate with arrow keys, press `enter` to drill into a ride, `q` to quit.

---

## Features

- **Three formats** — import `.fit`, `.gpx`, and `.tcx` files (Garmin, Wahoo, Strava exports, etc.)
- **Bulk import** — import an entire directory recursively; idempotent, skips duplicates automatically; `--dry-run` to preview before committing
- **Ride sorting** — sort the rides list by distance, duration, elevation, power, speed, or date with `--sort` and `--order`
- **Power curve** — mean maximal power (MMP) table and ASCII chart for any ride with power data; shows peak power at 5s, 30s, 1m, 5m, 10m, 20m, and 60m
- **Interactive TUI** — browse and paginate rides with a keyboard-driven interface (auto-detected when running in a terminal)
- **ASCII stream charts** — plot power, heart rate, speed, cadence, or altitude over time; overlay multiple fields on one chart with `--overlay`
- **Route map** — render the GPS route as a Unicode Braille map in the terminal; colour-coded by elevation when altitude data is present
- **Aggregated stats** — totals and averages by month, week, or year; year-over-year comparison with `--compare`
- **Personal records** — all-time bests for distance, duration, elevation, speed, power, HR, and more
- **Metric & imperial** — switch units with a single config command
- **JSON output** — pipe any command with `--json` for scripting and integrations
- **Fully local** — all data in `~/.paceline/data.db`; nothing leaves your machine

---

## Install

**Requirements:** Go 1.21+

```bash
go install github.com/HGClarke/paceline@latest
```

Or build from source:

```bash
git clone https://github.com/HGClarke/paceline.git
cd paceline
go build -o paceline .
```

---

## Quick Start

```bash
# 1. Import a single file or an entire directory
paceline import ~/Downloads/activities/

# 2. Browse your rides (interactive TUI in a terminal, plain table when piped)
paceline rides

# 3. Inspect a specific ride by its position number
paceline ride 1

# 4. Chart a sensor stream
paceline ride 1 stream --field=hr
```

---

## Commands

### `paceline import <file|directory>`

Parse and store ride files into the local database.

```bash
# Single file
paceline import morning_ride.fit

# Entire directory (recursive by default)
paceline import ~/Downloads/strava_export/

# Non-recursive — top-level files only
paceline import ~/Downloads/strava_export/ --no-recursive

# Preview what would be imported without actually importing
paceline import ~/Downloads/strava_export/ --dry-run
```

- Supports `.fit`, `.gpx`, `.tcx`
- Recursively finds all supported files in subdirectories by default
- Idempotent: re-importing the same filename is always a no-op
- Summary shows `imported`, `already exist`, and `errors` counts separately
- Skipped files (unsupported format, parse error) are reported on stderr

**Flags:**

| Flag | Description |
|------|-------------|
| `--no-recursive` | Only read top-level files; do not descend into subdirectories |
| `--dry-run` | Show how many files would be imported without importing anything |

---

### `paceline rides`

List rides, newest first. Launches an interactive TUI when running in a terminal.

```bash
paceline rides                          # 10 most recent
paceline rides --year=2025              # all rides in 2025
paceline rides --year=2025 --month=6    # June 2025 only
paceline rides --date=2025-06-15        # a specific day
paceline rides --from 2025-01-01 --to 2025-03-31
paceline rides --from 2025-06-01                 # on or after
paceline rides --to 2025-06-30                   # on or before
paceline rides --page=2 --limit=20      # pagination

# Sorting
paceline rides --sort distance          # longest first (desc by default)
paceline rides --sort duration
paceline rides --sort elevation
paceline rides --sort power             # avg power; rides without power sort last
paceline rides --sort speed
paceline rides --sort date              # default behavior

# Sort direction
paceline rides --sort distance --order asc   # shortest first
paceline rides --sort distance --order desc  # longest first (default)

# Combine with filters
paceline rides --year 2025 --sort elevation --limit 5
```

**TUI controls (interactive mode):**

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `n` / `→` | Next page |
| `p` / `←` | Previous page |
| `enter` | Show ride detail |
| `q` / `esc` | Quit |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--year` | — | Filter by year (e.g. `2025`) |
| `--month` | — | Filter by month `1–12` (defaults to current year if year omitted) |
| `--date` | — | Filter by exact date `YYYY-MM-DD` |
| `--from` | — | Filter rides on or after this date `YYYY-MM-DD` |
| `--to` | — | Filter rides on or before this date `YYYY-MM-DD` |
| `--page` | `1` | Page number |
| `--limit` | `10` | Results per page |
| `--sort` | `date` | Sort field: `date`, `distance`, `duration`, `elevation`, `power`, `speed` |
| `--order` | `desc` | Sort direction: `desc` or `asc` |

---

### `paceline ride <position>`

Show the full summary for a single ride. `<position>` is the `#` number shown in the rides list.

```bash
paceline ride 3
paceline ride 3 --json
```

---

### `paceline ride <position> power-curve`

Show the mean maximal power (MMP) curve for a ride. Requires power stream data.

```bash
paceline ride 3 power-curve
```

Output: a table of peak power at 7 canonical durations, followed by an ASCII chart sampled at ~50 log-spaced intervals for a smooth exponential curve.

```
Power Curve — Ride 3 (2025-04-05)

  DURATION │ POWER
  ─────────┼───────
  5s       │ 812 W
  30s      │ 634 W
  1 min    │ 521 W
  5 min    │ 380 W
  10 min   │ 342 W
  20 min   │ 298 W
  60 min   │ 261 W

 812 ┤╮
 ...
      5s  30s  1m   5m   10m  20m  1h
               power curve
```

Durations longer than the ride are automatically omitted from the table and chart.

---

### `paceline ride <position> route`

Render the GPS route as a Braille map in the terminal. Requires GPS data (lat/lon streams).

```bash
paceline ride 3 route
```

The map is drawn using Unicode Braille characters for sub-character resolution, with an ANSI colour gradient from dark green (low elevation) to red (high elevation) when altitude data is present. A legend is printed below the map.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | `78` | Character width of the map |
| `--height` | `28` | Character height of the map |

---

### `paceline ride <position> stream`

Render an ASCII line chart for a ride's time-series sensor data.

```bash
paceline ride 3 stream                            # auto-selects best available field
paceline ride 3 stream --field=hr
paceline ride 3 stream --field=power
paceline ride 3 stream --field=speed,altitude     # separate colored charts
paceline ride 3 stream --field=power --field=hr --overlay  # single overlaid chart
```

Available fields: `power`, `hr`, `speed`, `cadence`, `altitude`

When `--field` is omitted, the field is auto-selected by priority: **power → hr → speed**.

All charts are rendered in color. Multiple fields without `--overlay` print separate charts sequentially. With `--overlay`, all fields are rendered on a single chart with a per-series color legend.

---

### `paceline stats`

Show aggregated totals (ride count, distance, duration, elevation).

```bash
paceline stats                  # all-time totals (default)
paceline stats --year=2025
paceline stats --year=2025 --month=3
paceline stats --year=2025 --week=12
paceline stats --from 2025-01-01 --to 2025-03-31
paceline stats --from 2025-01-01               # open-ended range
paceline stats --json

# Year-over-year comparison (defaults to current year vs 2024)
paceline stats --compare 2024

# Explicit primary year
paceline stats --year 2025 --compare 2024

# Same month, different years
paceline stats --month 5 --year 2025 --compare 2024
```

---

### `paceline records`

Show personal records (all-time bests across 9 categories).

```bash
paceline records                        # all-time records
paceline records --year=2025            # records within 2025
paceline records --year=2025 --month=6  # records within June 2025
paceline records --year=2025 --week=12  # records within a specific ISO week
paceline records --from 2025-01-01 --to 2025-03-31
paceline records --from 2025-06-01
paceline records --json
```

Categories: longest distance, longest duration, most elevation gain, highest avg power, highest avg speed, highest max speed, highest avg HR, most calories, highest altitude.

---

### `paceline delete`

Delete rides from the database.

```bash
paceline delete ride 5          # delete ride #5 (prompts for confirmation)
paceline delete ride 5 --force  # skip confirmation
paceline delete all             # delete everything (prompts for confirmation)
paceline delete all --force
```

---

### `paceline config`

View and edit persistent configuration.

```bash
paceline config                     # show all settings
paceline config get units           # print a single value
paceline config set units imperial  # switch to imperial (mi, ft, mph)
paceline config set units metric    # switch back to metric (km, m, km/h)
```

Config is stored at `~/.paceline/config.toml`.

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (works with `rides`, `ride`, `stats`, `records`) |

---

## Configuration

| Key | Values | Default | Description |
|-----|--------|---------|-------------|
| `units` | `metric` \| `imperial` | `metric` | Distance, speed, and elevation units |

```toml
# ~/.paceline/config.toml
units = "imperial"
```

---

## Data Storage

| Path | Contents |
|------|----------|
| `~/.paceline/data.db` | DuckDB database (rides + streams) |
| `~/.paceline/config.toml` | User configuration |

The database is created automatically on first use. Original ride files are only read at import time and are never modified.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [cobra](https://github.com/spf13/cobra) | CLI framework |
| [go-duckdb](https://github.com/marcboeker/go-duckdb) | Embedded analytics database |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | Interactive TUI |
| [asciigraph](https://github.com/guptarohit/asciigraph) | Terminal line charts |
| [tablewriter](https://github.com/olekukonko/tablewriter) | Terminal tables |
| [tormoder/fit](https://github.com/tormoder/fit) | FIT file parser |

---

## Development

### Prerequisites

- **Go 1.21+**
- **golangci-lint** (one-time install):
  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```
  This installs the binary to `~/go/bin/`. Ensure that directory is on your `$PATH`, or the `Makefile` will fall back to the full path automatically.

### Common commands

```bash
# Run all checks (vet + tests + lint) — the recommended pre-commit command
make all

# Build the binary
make build

# Run tests
make test

# Lint only
make lint

# Run tests for a single package
go test ./internal/parser/...

# Run a single test by name
go test ./internal/parser/... -run TestParseFIT
```

Test data files live in `testdata/` (`sample.fit`, `sample.gpx`, `sample.tcx`).

---

## License

MIT
