# Stats All-Time Default

**Date:** 2026-05-27  
**Status:** Approved

## Problem

`paceline stats` with no flags defaults to the current month. This is unintuitive — a user running the command bare expects a summary of everything, not a silently-scoped slice of their data.

## Goal

When `paceline stats` is run with no flags, show all-time aggregated totals and label the output `"all time"`.

## Approach

Remove the `noFlags` injection block in `cmd/stats.go` that sets `f.Year` and `f.Month` to the current month when no flags are provided. The `buildStatsWhere` function already returns no `WHERE` clause for an empty `StatsFilters{}`, so `GetStats` naturally returns all-time data with no further changes.

Update the label for the no-flags case from `"current month"` to `"all time"`.

## Changes

### `cmd/stats.go` (only file modified)

**Remove** the `noFlags` block (lines 47–51) that injects current year/month:

```go
// REMOVE this block:
noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0
if noFlags {
    y, m := now.Year(), int(now.Month())
    f.Year = &y
    f.Month = &m
} else {
```

**Update** the label logic so no-flags → `"all time"`:

```go
var label string
if noFlags {
    label = "all time"
} else {
    // existing label-building logic
}
```

The `--month` without `--year` fallback (injects current year) is unchanged — it's correct behavior for explicit flag use.

No changes to `internal/store`, `internal/display`, or any other package.

## Non-Goals

- No changes to `--year`, `--month`, `--week` flag behavior.
- No new flags added.
- No TUI or JSON output format changes.

## Testing

Existing store and display tests are unaffected. The `cmd/stats.go` logic change should be covered by running `paceline stats` with no flags and confirming it returns all-time data with the `"all time"` label.
