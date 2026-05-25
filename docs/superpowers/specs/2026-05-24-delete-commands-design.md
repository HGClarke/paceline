# Delete Commands Design

**Date:** 2026-05-24  
**Status:** Approved

## Context

PacelineCLI has no way to remove data once imported. Users need two deletion capabilities:

1. Remove a single ride (and its associated stream data) by numeric ID
2. Wipe all stored data (truncate all tables) as a clean-slate reset

Both operations are destructive and irreversible, so they require a confirmation prompt by default, with a `--force` / `-f` flag to skip for scripting.

## Commands

```
paceline delete ride <id>    # Delete one ride + its streams (by numeric ID)
paceline delete all          # Truncate rides + streams tables entirely
```

Both subcommands support `-f` / `--force` to skip the confirmation prompt.

## Confirmation Behavior

- `delete ride <id>`: fetches and displays the ride summary, then prompts:  
  `Delete ride #42 (2024-03-15, 42.3 km)? [y/N]`
- `delete all`: queries current ride count, prompts:  
  `Delete all 17 rides? This cannot be undone. [y/N]`
- Typing anything except `y` / `Y` exits cleanly with "Cancelled." (exit 0)

## Architecture

### Store layer (`internal/store/rides.go`)

- `DeleteRide(id int64) error` — transaction: DELETE streams WHERE ride_id=id, then DELETE rides WHERE id=id; return "not found" error if 0 rides rows affected
- `DeleteAll() error` — transaction: DELETE FROM streams, DELETE FROM rides

### Command layer (`cmd/delete.go`)

- `deleteCmd` (parent: `paceline delete`) registered on `rootCmd`
- `deleteRideCmd` (`delete ride <id>`) — `cobra.ExactArgs(1)`, parse ID as int64, GetRide → confirm → DeleteRide
- `deleteAllCmd` (`delete all`) — `cobra.NoArgs`, get count via `GetStats` → confirm → DeleteAll
- `--force` / `-f` flag registered on each subcommand

## Error Handling

| Scenario | Behavior |
|---|---|
| Non-integer ID arg | `"invalid ride ID: must be a number"` |
| Ride not found | `"ride <id> not found"` (before prompt) |
| User cancels prompt | Print "Cancelled." — exit 0 |
| DB error in transaction | Roll back, bubble up — stderr + exit 1 |
| `delete all` with 0 rides | `"No rides to delete."` — exit 0 |

## FK Handling

`streams` has `REFERENCES rides(id)` with no CASCADE. Both delete methods delete streams first, then rides, within a transaction.

## Testing

- `TestDeleteRide` — insert ride + streams, DeleteRide, verify GetRide returns not-found and GetStreams returns empty
- `TestDeleteRide_NotFound` — DeleteRide on non-existent ID, expect error containing "not found"
- `TestDeleteAll` — insert 2 rides with streams, DeleteAll, verify ListRides returns 0 rides
