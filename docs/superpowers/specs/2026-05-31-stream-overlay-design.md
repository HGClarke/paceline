# Stream Overlay Design

**Date:** 2026-05-31
**Status:** Approved

## Overview

Add a `--overlay` flag to the `ride <id> stream` command that renders multiple fields on a single overlaid ASCII chart instead of separate sequential charts. All charts (single and overlay) gain color and drop the point count from their captions.

## Flag

- Name: `--overlay`
- Type: `bool`
- Default: `false`
- Scope: `streamCmd` only
- Behavior: when `false`, multiple fields print as separate charts (existing behavior); when `true`, all fields are rendered on one overlaid chart via `asciigraph.PlotMany`

## Display Layer

### Signature change

```go
// Before
func PrintStreamChart(w io.Writer, points []parser.Stream, field string)

// After
func PrintStreamChart(w io.Writer, allSeries [][]parser.Stream, fields []string)
```

Each element of `allSeries` corresponds to the stream data for the field at the same index in `fields`. In single-field mode the caller passes `[][]parser.Stream{points}` and `[]string{field}`.

### Single-field rendering (`len(fields) == 1`)

- Calls `asciigraph.Plot`
- Caption: the field name only (e.g. `"power"`)
- Color: field's assigned color from palette

### Overlay rendering (`len(fields) > 1`)

- Calls `asciigraph.PlotMany`
- Caption: `"stream overlay"`
- `SeriesLegends`: field names in order
- `SeriesColors`: each field's assigned color from palette

### Color palette

Fixed mapping applied consistently in both single and overlay mode:

| Field    | Color  |
|----------|--------|
| power    | Red    |
| hr       | Blue   |
| speed    | Green  |
| cadence  | Yellow |
| altitude | Cyan   |

## Command Layer (`cmd/stream.go`)

- Add `--overlay` bool flag (default `false`)
- When `overlay == false`: loop over fields, fetch `points` per field via `GetStreams`, call `PrintStreamChart(w, [][]parser.Stream{points}, []string{field})` per iteration
- When `overlay == true`: fetch stream data for each field into a `[][]parser.Stream`, call `PrintStreamChart(w, allSeries, fields)` once

## Caption Summary

| Mode         | Caption          |
|--------------|------------------|
| Single field | `"<field name>"` |
| Overlay      | `"stream overlay"` |

Point count is removed from all captions.

## Out of Scope

- Per-field y-axis scaling (overlay shares a single y-axis)
- `--overlay` flag on any command other than `stream`
- Color configuration by the user
