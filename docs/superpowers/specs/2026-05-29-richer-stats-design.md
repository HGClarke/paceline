# Richer Stats — Design Spec

**Date:** 2026-05-29  
**Feature:** #5 from roadmap — Richer Stats (Averages & Maximums)  
**Status:** Approved

---

## Overview

Expand `paceline stats` to include per-period averages and maximums for speed, power, and heart rate, in addition to the existing ride count and totals. All data already lives in the `rides` table; this is a SQL aggregation extension and a display expansion.

---

## Scope

**In scope:**
- `AVG(avg_speed_mps)`, `MAX(max_speed_mps)` — always shown (speed is never null)
- `AVG(avg_power_w)`, `MAX(max_power_w)` — shown only when at least one ride in the period has power data
- `AVG(avg_hr_bpm)`, `MAX(max_hr_bpm)` — shown only when at least one ride in the period has HR data
- JSON output updated to include new fields
- Respects existing `--year`, `--month`, `--week`, `--from`, `--to` filters unchanged

**Out of scope:**
- No new flags or commands
- No changes to `rides`, `ride`, or `records` commands
- No schema migrations (all columns already exist)

---

## Store Layer

### `Stats` struct changes (`internal/store/rides.go`)

Add six new fields. Speed is always present, so plain `float64`. Power and HR are nullable (nil = no rides in the period had that sensor):

```go
type Stats struct {
    RideCount       int
    TotalDistanceM  float64
    TotalDurationS  int
    TotalElevationM float64
    AvgSpeedMPS     float64
    MaxSpeedMPS     float64
    AvgPowerW       *float64
    MaxPowerW       *float64
    AvgHRBPM        *float64
    MaxHRBPM        *float64
}
```

### `GetStats()` SQL changes

Replace the existing 4-column `SELECT` with a 10-column query:

```sql
SELECT
    COUNT(*),
    COALESCE(SUM(distance_m), 0),
    COALESCE(SUM(duration_s), 0),
    COALESCE(SUM(elevation_gain_m), 0),
    COALESCE(AVG(avg_speed_mps), 0),
    COALESCE(MAX(max_speed_mps), 0),
    AVG(avg_power_w),
    MAX(max_power_w),
    AVG(avg_hr_bpm),
    MAX(max_hr_bpm)
FROM rides<where>
```

Speed uses `COALESCE(..., 0)` (non-nullable). Power and HR scan into `sql.NullFloat64` and are set to `nil` on the `Stats` struct when `!Valid`.

**Semantics:**
- `Avg speed` = `AVG(avg_speed_mps)` — average of per-ride averages across the period
- `Max speed` = `MAX(max_speed_mps)` — highest single-ride peak speed in the period
- `Avg power` = `AVG(avg_power_w)` — average of per-ride averages (rides without power excluded by SQL `AVG` NULL semantics)
- `Max power` = `MAX(max_power_w)` — highest single-ride peak power in the period
- `Avg HR` / `Max HR` — same pattern as power

---

## Display Layer

### `PrintStats()` changes (`internal/display/table.go`)

Append speed rows unconditionally, then conditionally append power and HR rows if non-nil. Matches the existing conditional pattern in `PrintRideDetail`.

**Table output (metric, with power + HR):**
```
Stats: all time

  Rides            14
  Total Distance   423.7 km
  Total Duration   18h 42m 00s
  Total Elevation  4210 m
  Avg Speed        28.4 km/h
  Max Speed        52.1 km/h
  Avg Power        231 W
  Max Power        421 W
  Avg HR           148 bpm
  Max HR           183 bpm
```

**Table output (metric, no power/HR data):**
```
Stats: all time

  Rides            3
  Total Distance   87.2 km
  Total Duration   3h 14m 00s
  Total Elevation  820 m
  Avg Speed        27.0 km/h
  Max Speed        44.3 km/h
```

### JSON output

New fields appear in the JSON marshaling of `Stats`. Speed fields are always present. Power/HR fields are `null` when no data:

```json
{
  "RideCount": 14,
  "TotalDistanceM": 423700,
  "TotalDurationS": 67320,
  "TotalElevationM": 4210,
  "AvgSpeedMPS": 7.89,
  "MaxSpeedMPS": 14.47,
  "AvgPowerW": 231.4,
  "MaxPowerW": 421.0,
  "AvgHRBPM": 148.2,
  "MaxHRBPM": 183.0
}
```

---

## Testing

### Store tests (`internal/store/rides_test.go`)

- `TestGetStats_NewFields_WithSensorData` — insert rides with power + HR, assert all 10 fields populated correctly
- `TestGetStats_NewFields_NoSensorData` — insert rides without power/HR, assert `AvgPowerW`, `MaxPowerW`, `AvgHRBPM`, `MaxHRBPM` are nil; speed fields non-zero

### Display tests (`internal/display/display_test.go`)

- `TestPrintStats_WithPowerAndHR` — assert "Avg Power", "Max Power", "Avg HR", "Max HR", "Avg Speed", "Max Speed" appear in table output
- `TestPrintStats_NoPowerNoHR` — assert power/HR rows absent; speed rows present
- `TestPrintStats_JSON` — update existing test to assert new JSON keys present

---

## File Changelist

| File | Change |
|------|--------|
| `internal/store/rides.go` | Extend `Stats` struct; update `GetStats()` SQL and scan |
| `internal/display/table.go` | Update `PrintStats()` to render new rows conditionally |
| `internal/store/rides_test.go` | Add 2 new store tests |
| `internal/display/display_test.go` | Update existing JSON test; add 2 new display tests |
