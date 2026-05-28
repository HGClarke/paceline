# Personal Records — Design Spec

**Date:** 2026-05-27  
**Status:** Approved

---

## Overview

A `paceline records` command that queries the database for all-time personal bests across 9 categories. Defaults to all-time but supports the same time-period filters as `stats`. Output is a dynamic table that omits records for which no data exists.

---

## Records

| Record | Source column | Table | Nullable |
|---|---|---|---|
| Longest distance | `distance_m` | `rides` | No |
| Longest duration | `duration_s` | `rides` | No |
| Most elevation gain | `elevation_gain_m` | `rides` | No |
| Highest avg power | `avg_power_w` | `rides` | Yes |
| Highest avg speed | `avg_speed_mps` | `rides` | No |
| Highest avg HR | `avg_hr_bpm` | `rides` | Yes |
| Highest max speed | `max_speed_mps` | `rides` | No |
| Most calories | `calories` | `rides` | Yes |
| Highest altitude point | `MAX(altitude_m)` | `streams` (joined to `rides`) | Yes |

---

## Command

```bash
paceline records                  # all time (default)
paceline records --year 2025      # best within 2025
paceline records --month 5        # best in May of current year
paceline records --week 22        # best in ISO week 22 of current year
paceline records --json           # JSON output
```

Flags mirror `stats` exactly. `--json` is inherited from `rootCmd`.

---

## Architecture

### Store layer — `internal/store/rides.go`

**New types:**

```go
type RecordsFilters struct {
    Year  *int
    Month *int
    Week  *int
}

// PersonalRecord holds the value and date of one all-time best.
// A nil pointer means no data exists for that record category.
type PersonalRecord struct {
    RawValue float64
    Date     time.Time
}

type Records struct {
    LongestDistanceM   *PersonalRecord
    LongestDurationS   *PersonalRecord
    MostElevationGainM *PersonalRecord
    HighestAvgPowerW   *PersonalRecord
    HighestAvgSpeedMPS *PersonalRecord
    HighestAvgHRBPM    *PersonalRecord
    HighestMaxSpeedMPS *PersonalRecord
    MostCaloriesKcal   *PersonalRecord
    HighestAltitudeM   *PersonalRecord
}
```

**`GetRecords(f RecordsFilters) (Records, error)`:**

Runs 9 sequential queries. Each query returns at most one row.

For the 8 rides-table records, the query pattern is:

```sql
SELECT <field>, recorded_at
FROM rides
[WHERE year/month/week filters]
[AND <field> IS NOT NULL]   -- only for nullable fields
ORDER BY <field> DESC
LIMIT 1
```

The WHERE clause is built by a new `buildRecordsWhere` helper (identical logic to `buildStatsWhere` — year, month, week extraction against `recorded_at`).

For the altitude record:

```sql
SELECT MAX(s.altitude_m), r.recorded_at
FROM streams s
JOIN rides r ON r.id = s.ride_id
[WHERE r.year/month/week filters]
GROUP BY r.id, r.recorded_at
ORDER BY MAX(s.altitude_m) DESC
LIMIT 1
```

Each query scans into a `sql.NullFloat64` value and `sql.NullTime` date. A no-row result (or a NULL value scan) leaves the corresponding `Records` field as `nil`. Errors from any query propagate immediately, aborting `GetRecords`.

Queries run sequentially. DuckDB in-process MAX queries over the rides table complete in single-digit milliseconds each; parallelism would add goroutine/channel complexity for no perceptible user benefit.

---

### Display layer — `internal/display/table.go`

**New function `PrintRecords(w io.Writer, recs store.Records, label string, jsonOut bool, units string)`:**

**Table output:**

Renders a 3-column table (Record | Value | Date) using the same `tablewriter` border style as the rest of the CLI.

- Each row corresponds to one `*PersonalRecord` field.
- If a field is `nil`, that row is **omitted entirely** — no placeholder, no dash. This means a user who has never recorded HR sees no "Highest avg HR" row.
- If all 9 fields are nil (empty database or no rides match the filter), the table is skipped and a message is printed instead:
  ```
  No rides imported yet — run 'paceline import <file>' to get started.
  ```
  When a filter is active and all records are nil, the message is:
  ```
  No rides found for the selected period.
  ```

Value formatting reuses existing helpers:
- Distance → `FormatDistance(m, units)`
- Elevation gain, altitude → `FormatElevation(m, units)`
- Speed (avg, max) → `formatSpeed(mps, units)` (package-private; keep consistent)
- Duration → `formatDuration(int(rawValue))`
- Power → `fmt.Sprintf("%d W", int(rawValue))`
- HR → `fmt.Sprintf("%d bpm", int(rawValue))`
- Calories → `strconv.Itoa(int(rawValue))`

Date column: `date.Format("2006-01-02")`.

**Example output:**

```
Personal Records: all time

 Record               Value        Date
 ─────────────────── ──────────── ──────────
 Longest distance     142.3 km     2024-08-10
 Longest duration     5h 12m 04s   2024-08-10
 Most elevation gain  2,840 m      2023-07-04
 Highest avg power    287 W        2025-03-15
 Highest avg speed    38.4 km/h    2024-05-01
 Highest avg HR       172 bpm      2025-01-18
 Highest max speed    52.3 km/h    2024-05-01
 Most calories        1,840        2024-08-10
 Highest altitude     2,105 m      2023-07-04
```

If the user has never recorded power or HR, those two rows are absent.

**JSON output:**

Encodes `store.Records` directly. Nil `*PersonalRecord` fields serialise as `null`.

---

### Command layer — `cmd/records.go`

New file. Registers `recordsCmd` on `rootCmd`. Flags: `--year int`, `--month int`, `--week int`. Label-building logic identical to `cmd/stats.go`.

Flow:
1. Open store
2. Build `RecordsFilters` + label from flags
3. Call `s.GetRecords(f)`
4. Call `display.PrintRecords(os.Stdout, recs, label, jsonOutput, cfg.Units)`

---

## Testing

### Store — `internal/store/rides_test.go`

| Test | Description |
|---|---|
| `TestGetRecords_Empty` | Empty DB → all 9 fields nil |
| `TestGetRecords_AllFields` | One ride with all fields + stream rows → all 9 records populated with correct values and dates |
| `TestGetRecords_MissingNullable` | Rides with no power/HR/calories, no stream rows → those 4 fields nil; other 5 set |
| `TestGetRecords_YearFilter` | Rides across two years → `--year` returns only the correct year's best |
| `TestGetRecords_PicksMax` | Two rides where each field is best on a different ride → each record points to the correct date |

### Display — `internal/display/display_test.go`

| Test | Description |
|---|---|
| `TestPrintRecords_FullTable` | All 9 records present → all 9 rows rendered |
| `TestPrintRecords_PartialTable` | Nil nullable records → those rows absent from output |
| `TestPrintRecords_EmptyDB` | All nil → "No rides imported yet" message |
| `TestPrintRecords_EmptyFilter` | All nil with active filter → "No rides found for the selected period." |
| `TestPrintRecords_JSON` | JSON output → nil fields serialise as `null` |

---

## Files Changed

| File | Change |
|---|---|
| `internal/store/rides.go` | Add `RecordsFilters`, `PersonalRecord`, `Records`, `GetRecords`, `buildRecordsWhere` |
| `internal/display/table.go` | Add `PrintRecords` |
| `cmd/records.go` | New file — `recordsCmd` |
| `internal/store/rides_test.go` | Add 5 store tests |
| `internal/display/display_test.go` | Add 5 display tests |

No schema changes. No new dependencies.
