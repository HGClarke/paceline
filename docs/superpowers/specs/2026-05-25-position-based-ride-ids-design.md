# Position-Based Ride IDs

**Date:** 2026-05-25  
**Status:** Approved

## Problem

Rides are assigned a permanent integer ID from a DuckDB sequence at import time. After deleting rides and re-importing, the IDs shown in `rides` are no longer sequential (e.g. `1, 3, 4` after deleting ride 2, or starting at `5` after a full reset and re-import). This creates a mismatch between user expectation ("the first ride in the list is ride 1") and reality.

The primary user workflow is: run `rides`, spot a ride visually, then type `ride <n>`. There is no scripting or ID-memorisation use case.

## Decision

Replace user-facing DB IDs with **globally-computed position numbers**. Position is derived at query time using a SQL window function — it is never stored. The internal DB schema is unchanged.

- **Position 1** = most recently recorded ride (across all rides, no filter)
- Position is stable within a session: as long as no new rides are imported or deleted between `rides` and `ride <n>`, the number the user sees is the number they type
- Importing a ride whose date falls between existing rides will shift the positions of older rides. This is acceptable given the look-then-type workflow.

## No Migration Required

Position is computed via `ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC)` on every query. The `rides` table schema, sequence, and foreign keys are untouched. Existing databases work immediately with the new binary.

## Design

### Data model

`parser.Ride` gains a `Position int64` field. The existing `ID int64` field is kept for internal DB operations (FK lookups, stream fetching) but is never exposed in output.

```go
type Ride struct {
    ID       int64  // internal DB primary key — not shown to users
    Position int64  // display rank: 1 = most recent — what users type
    // ... existing fields unchanged
}
```

### Store layer (`internal/store/rides.go`)

**`ListRides`** wraps the query in a CTE to attach a global position to each row:

```sql
WITH ranked AS (
  SELECT *, ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC) AS position
  FROM rides
)
SELECT * FROM ranked
WHERE <filters>
ORDER BY recorded_at DESC, id DESC
LIMIT ? OFFSET ?
```

Filters (`--year`, `--month`, `--date`) are applied inside the outer query, after the global rank is computed. This means filtered views display each ride's **global** position, not a local rank within the filtered set. A ride that is 5th most recent globally will show as `#5` even if it is the only result in a filtered view. This is intentional: it ensures `ride 5` always resolves correctly regardless of how the user discovered the number.

**New: `GetRideByPosition(pos int64) (parser.Ride, error)`** — resolves a user-supplied position to a ride:

```sql
WITH ranked AS (
  SELECT *, ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC) AS position
  FROM rides
)
SELECT * FROM ranked WHERE position = ?
```

Returns a clear error (`"no ride at position N"`) when the position is out of range.

**`GetRide(id int64)`** is kept for internal use only (fetching a ride by DB ID after position resolution). Because it does not use the CTE, `Position` will be 0 on rides returned by this method — that is acceptable since it is never called for display.

**`scanRide`** needs updating to conditionally scan the `position` column. The simplest approach is a separate `scanRankedRide` helper used by `ListRides` and `GetRideByPosition`, while `scanRide` remains unchanged for `GetRide`.

### Command layer

**`ride <n>`** (`cmd/ride.go`):
- `PersistentPreRunE` parses `args[0]` as a position integer, opens the store, calls `GetRideByPosition(n)`, and stores the resolved DB ID in the existing `currentRideID` package variable.
- `runRide` and `streamCmd` continue using `currentRideID` as a DB ID — no changes needed there.

**`delete <n>`** (`cmd/delete.go`):
- Same pattern: parse position → resolve to DB ID via `GetRideByPosition` → delete by DB ID.

**Error messaging:** Both commands must produce user-friendly errors for out-of-range positions: `"no ride at position 3 — run 'paceline rides' to see available rides"`.

### Display (`internal/display/table.go`, `tui.go`)

- `PrintRideList`: rename column header `ID` → `#`; render `r.Position` instead of `r.ID`
- `PrintRideDetail`: position not shown (user is already viewing the ride)
- `RunRidesTUI`: show `#` column with position
- **JSON output**: the `position` field is the user-facing identifier in ride JSON objects. The internal `id` field is omitted — it is an implementation detail with no external value at this time.

### Tie-breaking

The window function uses `ORDER BY recorded_at DESC, id DESC` as a secondary sort. This ensures rides with identical timestamps always appear in a consistent, deterministic order.

## Trade-offs

| Scenario | Behaviour |
|---|---|
| Import 3 rides in any order | List always shows `#1, #2, #3` sorted by ride date |
| Delete ride #2 | Remaining rides recompute to `#1, #2` — clean |
| Import a new ride (newest date) | New ride becomes `#1`; others shift down by 1 |
| Import a ride that slots in the middle | Rides older than the new one shift down by 1 |
| Filtered view (`--month 3`) | Shows global positions; may be non-sequential within the filtered set |

Position shifts on import are acceptable because the user always looks at the list before typing a position number.

## Testing

- **`store/rides_test.go`**: add tests for `GetRideByPosition` — happy path, out-of-range error, correct position after delete, position shift when a mid-date ride is inserted
- **`display/display_test.go`**: update column header assertions from `ID` → `#`; verify `Position` is rendered, not `ID`
- **`cmd/` tests**: update any tests that pass DB IDs to `ride` or `delete` to pass position numbers instead
