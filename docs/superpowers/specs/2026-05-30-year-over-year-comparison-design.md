# Year-over-Year Comparison — Design Spec

**Date:** 2026-05-30  
**Status:** Approved  
**Feature:** `paceline stats --compare <year>`

---

## Overview

Add a `--compare <year>` flag to the `stats` command that runs two `GetStats` queries and renders them side-by-side with a delta column. No new data or schema changes are required — this is purely a flag + display layer addition.

---

## Command Interface

```bash
# Compare current year to 2024 (--year defaults to current year)
paceline stats --compare 2024

# Explicit primary year
paceline stats --year 2025 --compare 2024

# Same month in different years
paceline stats --month 5 --year 2025 --compare 2024

# Compare any two years
paceline stats --year 2024 --compare 2023

# JSON output
paceline stats --year 2025 --compare 2024 --json
```

**Flag:** `--compare int` on `statsCmd`. Default `0` (disabled).

**Defaults and validation:**
- If `--year` is not set when `--compare` is used, year defaults to the current year (same behaviour as `--month` already infers the current year).
- If `--compare` equals the resolved primary year → error: `--compare year must differ from the primary year`.
- If combined with `--week`, `--from`, or `--to` → error: `--compare is only supported with --year and --month`.

---

## Store Layer

No changes. `GetStats(StatsFilters)` is called twice sequentially:
1. Primary call: same filters as today (`Year`, `Month` if set).
2. Compare call: same `Month` (if set), `Year` = compare year.

DuckDB is embedded and synchronous; no goroutines needed.

---

## Display Layer

New function in `internal/display/table.go`:

```go
func PrintStatsComparison(w io.Writer, st1, st2 store.Stats, label1, label2 string, jsonOut bool, units string)
```

### Table output

```
Stats: 2025 vs 2024

  METRIC            2025        2024        Δ
  Rides             87          112         -25 (-22%)
  Total Distance    2,341.0 km  4,107.3 km  -1,766.3 km (-43%)
  Total Duration    96h 14m     178h 02m    -81h 48m (-46%)
  Total Elevation   21,840 m    38,920 m    -17,080 m (-44%)
```

**Metrics included:** Rides, Total Distance, Total Duration, Total Elevation (totals only — richer avg/max metrics are excluded).

**Delta formatting:**
- Positive: `+value (+pct%)`
- Negative: `-value (-pct%)`
- Compare year has 0 rides (division by zero): show delta value only, no percentage (e.g. `+87`)

**Label construction:**
- Year only: `"2025 vs 2024"`
- Year + Month: `"May 2025 vs May 2024"` (using `time.Month(m).String()`)

### JSON output

```json
{
  "primary": { "label": "2025", "stats": { ... } },
  "compare": { "label": "2024", "stats": { ... } }
}
```

`stats` fields are the full `store.Stats` struct (same as single-year JSON).

---

## Files Changed

| File | Change |
|------|--------|
| `cmd/stats.go` | Add `--compare` flag, validation logic, second `GetStats` call, route to `PrintStatsComparison` |
| `internal/display/table.go` | Add `PrintStatsComparison` function |
| `internal/display/display_test.go` | Add tests for `PrintStatsComparison` (table and JSON) |

---

## Out of Scope

- `--week` and `--from/--to` filter combinations with `--compare`
- Richer metrics (avg/max speed, power, HR) in the comparison table
- A standalone `compare` subcommand
