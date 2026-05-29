---
title: Pretty-Print JSON Output
date: 2026-05-28
status: approved
---

## Overview

When the `--json` flag is set, all JSON output is currently emitted as compact single-line JSON via `json.NewEncoder`. This change switches to indented, human-readable JSON using `json.MarshalIndent`.

## Change

**File:** `internal/display/table.go`

Replace all 4 `json.NewEncoder(w).Encode(x)` calls with:

```go
b, _ := json.MarshalIndent(x, "", "  ")
fmt.Fprintln(w, string(b))
```

**Affected functions:**
- `PrintRideList`
- `PrintRideDetail`
- `PrintStats`
- `PrintRecords`

**Indent:** 2 spaces.

**Error handling:** `MarshalIndent` error is discarded (`_`) — consistent with the existing pattern, since a write error to stdout is unrecoverable.

## Out of Scope

- No new flags (e.g. `--compact`) to toggle between formats.
- No changes to any other files.

## Tests

Existing JSON tests in `internal/display/display_test.go` assert on key/value presence via `strings.Contains`, so they pass unchanged. No new tests needed.
