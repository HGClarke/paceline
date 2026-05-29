# Date Range Filtering — Design Spec

**Date:** 2026-05-28  
**Status:** Approved  
**Feature roadmap item:** #3

---

## Overview

Add `--from` and `--to` flags (accepting `YYYY-MM-DD`) to the `rides`, `stats`, and `records` commands, allowing users to filter by an arbitrary date range rather than being limited to calendar-aligned year/month/week boundaries.

**Out of scope (noted for future):** Shorthand keyword aliases (`--range last-30-days`, `--range ytd`, `--range last-week`). Deferred to a follow-on feature.

---

## Architecture

The change touches two layers:

1. **Store layer** — filter structs + SQL clause building (`internal/store/rides.go`)
2. **Command layer** — flag definitions + input parsing + label building (`cmd/rides.go`, `cmd/stats.go`, `cmd/records.go`)

No schema changes. No display changes.

---

## Store Layer (`internal/store/rides.go`)

### Filter structs

Add `From *time.Time` and `To *time.Time` to all three filter structs:

```go
type RideFilters struct {
    Year  *int
    Month *int
    Date  *time.Time
    From  *time.Time  // new
    To    *time.Time  // new
    Page  int
    Limit int
}

type StatsFilters struct {
    Year  *int
    Month *int
    Week  *int
    From  *time.Time  // new
    To    *time.Time  // new
}

type RecordsFilters struct {
    Year  *int
    Month *int
    Week  *int
    From  *time.Time  // new
    To    *time.Time  // new
}
```

### Shared helper

```go
func appendDateRangeClauses(clauses []string, args []any, from, to *time.Time) ([]string, []any) {
    if from != nil {
        clauses = append(clauses, "recorded_at >= ?::DATE")
        args = append(args, from.Format("2006-01-02"))
    }
    if to != nil {
        clauses = append(clauses, "recorded_at < (?::DATE + INTERVAL 1 DAY)")
        args = append(args, to.Format("2006-01-02"))
    }
    return clauses, args
}
```

The `+1 day` on `To` makes the upper bound inclusive: `--to 2025-03-31` includes all rides on March 31.

### Each `buildXxxWhere` function

Each function calls `appendDateRangeClauses` after its existing Year/Month/Week/Date clauses. All active filters AND together.

---

## Command Layer

### New flags (all three commands)

```
--from   string   filter rides on or after this date (YYYY-MM-DD)
--to     string   filter rides on or before this date (YYYY-MM-DD)
```

### Parsing and validation

- Parse with `time.Parse("2006-01-02", ...)`, same pattern as existing `--date` in `rides.go`
- Clear error on bad format: `invalid --from %q: use YYYY-MM-DD`
- If both are provided and `--from` is after `--to`: return error `--from must not be after --to`
- Open-ended ranges are valid: `--from` only, or `--to` only

### Interaction with existing flags

`--from`/`--to` are additive with `--year`/`--month`/`--week`/`--date` — all active filters AND together in SQL. No mutual exclusion enforced.

### Label building (`stats` and `records`)

Extend the existing label-building logic:

| Active flags | Label |
|---|---|
| `--from 2025-01-01 --to 2025-03-31` | `2025-01-01 to 2025-03-31` |
| `--from 2025-01-01` only | `from 2025-01-01` |
| `--to 2025-03-31` only | `to 2025-03-31` |
| Combined with year: `--year 2025 --from 2025-01-01` | `2025 from 2025-01-01` |

---

## Testing (`internal/store/`)

New test cases in `rides_test.go`, and equivalent coverage in `streams_test.go` (for `RecordsFilters` via altitude records) and the stats/records test files:

| Test | What it verifies |
|---|---|
| `--from` only | Rides on or after the date are returned; earlier rides excluded |
| `--to` only | Rides on or before the date (inclusive) are returned; later rides excluded |
| `--from` + `--to` | Only rides within the range returned |
| Range boundary | A ride exactly on `--from` date is included; one exactly on `--to` date is included |
| `--from`/`--to` + `--year` | Both filters AND together correctly |

Validation (`--from` after `--to`) is tested at the cmd layer implicitly through the error return — no store-level test needed.

---

## Future Work

- Shorthand keyword aliases: `--range last-30-days`, `--range last-week`, `--range ytd`, `--range last-90-days`. Would be parsed in the cmd layer and expanded to `From`/`To` values before passing to the store. No store changes required.
