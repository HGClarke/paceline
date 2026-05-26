# Position-Based Ride IDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace user-facing DB sequence IDs with globally computed position numbers (1 = most recent ride) so the rides list always shows clean sequential identifiers after deletes and re-imports.

**Architecture:** Position is computed at query time using `ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC)` — never stored. The `rides` table schema is untouched. `parser.Ride` gains a `Position int64` field; the internal `ID` field remains for FK lookups but is hidden from output. Commands resolve a user-supplied position to a DB ID via a new `GetRideByPosition` store method before doing any further work.

**Tech Stack:** Go, DuckDB (`go-duckdb`), Cobra, tablewriter, bubbletea

---

## File Map

| File | Change |
|---|---|
| `internal/parser/model.go` | Add `Position int64` and JSON tags to `Ride` |
| `internal/store/rides.go` | Add `GetRideByPosition`, `scanRankedRide`; update `ListRides` CTE |
| `internal/store/rides_test.go` | New tests for `GetRideByPosition` and `ListRides` positions |
| `internal/display/table.go` | Rename `ID` → `#`; render `r.Position` |
| `internal/display/display_test.go` | Add `TestPrintRideList_ShowsPositionColumn` |
| `cmd/ride.go` | `PersistentPreRunE` resolves position → DB ID |
| `cmd/delete.go` | `runDeleteRide` resolves position → DB ID |
| `internal/display/tui.go` | Rename `ID` header → `#`; render `r.Position` |

---

## Task 1: Add `Position` field and JSON tags to `parser.Ride`

**Files:**
- Modify: `internal/parser/model.go`

- [ ] **Step 1: Update `parser.Ride`**

Replace the existing struct in `internal/parser/model.go`:

```go
package parser

import "time"

type Ride struct {
	ID             int64     `json:"-"`
	Position       int64     `json:"id"`
	Filename       string    `json:"filename"`
	RecordedAt     time.Time `json:"recorded_at"`
	DistanceM      float64   `json:"distance_m"`
	DurationS      int       `json:"duration_s"`
	ElevationGainM float64   `json:"elevation_gain_m"`
	AvgSpeedMPS    float64   `json:"avg_speed_mps"`
	MaxSpeedMPS    float64   `json:"max_speed_mps"`
	AvgHRBPM       *int      `json:"avg_hr_bpm,omitempty"`
	MaxHRBPM       *int      `json:"max_hr_bpm,omitempty"`
	AvgPowerW      *int      `json:"avg_power_w,omitempty"`
	MaxPowerW      *int      `json:"max_power_w,omitempty"`
	Calories       *int      `json:"calories,omitempty"`
	SourceFormat   string    `json:"source_format"`
}

type Stream struct {
	RideID     int64
	Timestamp  time.Time
	ElapsedS   int
	SpeedMPS   *float64
	HRBPM      *int
	PowerW     *int
	CadenceRPM *int
	AltitudeM  *float64
	Lat        *float64
	Lon        *float64
}
```

`ID` is tagged `json:"-"` (omitted from JSON output). `Position` is tagged `json:"id"` (exposed as `"id"` in JSON so the user-facing key matches what they type).

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: no errors. (All existing code referencing `r.ID` still compiles — we have not removed the field.)

- [ ] **Step 3: Commit**

```bash
git add internal/parser/model.go
git commit -m "feat: add Position field to parser.Ride; omit internal ID from JSON"
```

---

## Task 2: Add `GetRideByPosition` and `scanRankedRide` to the store

**Files:**
- Modify: `internal/store/rides.go`
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/store/rides_test.go` (after the last existing test):

```go
func TestGetRideByPosition_HappyPath(t *testing.T) {
	s := openTestStore(t)

	idNewest, err := s.InsertRide(parser.Ride{
		Filename:     "newest.gpx",
		RecordedAt:   time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert newest: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{
		Filename:     "oldest.gpx",
		RecordedAt:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert oldest: %v", err)
	}

	got, err := s.GetRideByPosition(1)
	if err != nil {
		t.Fatalf("GetRideByPosition(1): %v", err)
	}
	if got.ID != idNewest {
		t.Errorf("position 1: got DB ID %d, want %d", got.ID, idNewest)
	}
	if got.Position != 1 {
		t.Errorf("Position field: got %d, want 1", got.Position)
	}
}

func TestGetRideByPosition_OutOfRange(t *testing.T) {
	s := openTestStore(t)
	_, err := s.InsertRide(parser.Ride{
		Filename:     "only.gpx",
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, err = s.GetRideByPosition(99)
	if err == nil {
		t.Fatal("expected error for out-of-range position, got nil")
	}
	if !strings.Contains(err.Error(), "position 99") {
		t.Errorf("expected error to mention position 99, got: %v", err)
	}
}

func TestGetRideByPosition_AfterDelete(t *testing.T) {
	s := openTestStore(t)

	idA, err := s.InsertRide(parser.Ride{Filename: "a.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert a: %v", err)
	}
	idB, err := s.InsertRide(parser.Ride{Filename: "b.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert b: %v", err)
	}
	idC, err := s.InsertRide(parser.Ride{Filename: "c.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert c: %v", err)
	}
	_ = idA

	// Before delete: position 2 is B
	got, err := s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) before delete: %v", err)
	}
	if got.ID != idB {
		t.Errorf("before delete: position 2 got ID %d, want %d (B)", got.ID, idB)
	}

	if err := s.DeleteRide(idB); err != nil {
		t.Fatalf("DeleteRide: %v", err)
	}

	// After delete: position 2 is now C
	got, err = s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) after delete: %v", err)
	}
	if got.ID != idC {
		t.Errorf("after delete: position 2 got ID %d, want %d (C)", got.ID, idC)
	}
}

func TestGetRideByPosition_PositionShiftOnMidInsert(t *testing.T) {
	s := openTestStore(t)

	idRecent, err := s.InsertRide(parser.Ride{Filename: "recent.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert recent: %v", err)
	}
	idOld, err := s.InsertRide(parser.Ride{Filename: "old.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert old: %v", err)
	}
	_ = idRecent

	// Before mid-insert: position 2 is old
	got, err := s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) before mid-insert: %v", err)
	}
	if got.ID != idOld {
		t.Errorf("before mid-insert: position 2 got ID %d, want %d (old)", got.ID, idOld)
	}

	// Insert a ride whose date falls between the two
	_, err = s.InsertRide(parser.Ride{Filename: "mid.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert mid: %v", err)
	}

	// After mid-insert: position 2 is mid, position 3 is old
	got, err = s.GetRideByPosition(2)
	if err != nil {
		t.Fatalf("GetRideByPosition(2) after mid-insert: %v", err)
	}
	if got.Filename != "mid.gpx" {
		t.Errorf("after mid-insert: position 2 filename = %q, want \"mid.gpx\"", got.Filename)
	}

	got, err = s.GetRideByPosition(3)
	if err != nil {
		t.Fatalf("GetRideByPosition(3) after mid-insert: %v", err)
	}
	if got.ID != idOld {
		t.Errorf("after mid-insert: position 3 got ID %d, want %d (old)", got.ID, idOld)
	}
}
```

- [ ] **Step 2: Run tests — verify they all fail**

```bash
go test ./internal/store/... -run "TestGetRideByPosition" -v
```

Expected: FAIL with `s.GetRideByPosition undefined`.

- [ ] **Step 3: Add `scanRankedRide` and `GetRideByPosition` to `internal/store/rides.go`**

Add these two functions at the bottom of `internal/store/rides.go`, after the existing `scanRide` function:

```go
// scanRankedRide is like scanRide but also scans the position column.
// Use this with queries that include a ROW_NUMBER() position column.
func scanRankedRide(row rideScanner) (parser.Ride, error) {
	var r parser.Ride
	var avgHR, maxHR, avgPower, maxPower, calories sql.NullInt64
	err := row.Scan(
		&r.ID, &r.Filename, &r.RecordedAt,
		&r.DistanceM, &r.DurationS, &r.ElevationGainM,
		&r.AvgSpeedMPS, &r.MaxSpeedMPS,
		&avgHR, &maxHR, &avgPower, &maxPower, &calories,
		&r.SourceFormat,
		&r.Position,
	)
	if err != nil {
		return r, fmt.Errorf("scan ride: %w", err)
	}
	if avgHR.Valid {
		v := int(avgHR.Int64)
		r.AvgHRBPM = &v
	}
	if maxHR.Valid {
		v := int(maxHR.Int64)
		r.MaxHRBPM = &v
	}
	if avgPower.Valid {
		v := int(avgPower.Int64)
		r.AvgPowerW = &v
	}
	if maxPower.Valid {
		v := int(maxPower.Int64)
		r.MaxPowerW = &v
	}
	if calories.Valid {
		v := int(calories.Int64)
		r.Calories = &v
	}
	return r, nil
}

// GetRideByPosition returns the ride at the given global position (1 = most recent).
// Returns an error if the position is out of range.
func (s *Store) GetRideByPosition(pos int64) (parser.Ride, error) {
	row := s.db.QueryRow(`
		WITH ranked AS (
			SELECT id, filename, recorded_at, distance_m, duration_s,
				elevation_gain_m, avg_speed_mps, max_speed_mps,
				avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
				calories, source_format,
				ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC) AS position
			FROM rides
		)
		SELECT id, filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format, position
		FROM ranked WHERE position = ?`, pos)
	r, err := scanRankedRide(row)
	if errors.Is(err, sql.ErrNoRows) {
		return parser.Ride{}, fmt.Errorf("no ride at position %d — run 'paceline rides' to see available rides", pos)
	}
	return r, err
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/store/... -run "TestGetRideByPosition" -v
```

Expected: all 4 `TestGetRideByPosition_*` tests PASS.

- [ ] **Step 5: Run full test suite — verify nothing is broken**

```bash
go test ./...
```

Expected: all existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat: add GetRideByPosition and scanRankedRide to store"
```

---

## Task 3: Update `ListRides` to compute and return global positions

**Files:**
- Modify: `internal/store/rides.go`
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Write a failing test**

Append to `internal/store/rides_test.go`:

```go
func TestListRides_HasPositions(t *testing.T) {
	s := openTestStore(t)

	_, err := s.InsertRide(parser.Ride{Filename: "newest.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert newest: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{Filename: "middle.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert middle: %v", err)
	}
	_, err = s.InsertRide(parser.Ride{Filename: "oldest.gpx", RecordedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"})
	if err != nil {
		t.Fatalf("insert oldest: %v", err)
	}

	rides, _, err := s.ListRides(store.RideFilters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if len(rides) != 3 {
		t.Fatalf("expected 3 rides, got %d", len(rides))
	}
	if rides[0].Position != 1 {
		t.Errorf("rides[0].Position = %d, want 1", rides[0].Position)
	}
	if rides[1].Position != 2 {
		t.Errorf("rides[1].Position = %d, want 2", rides[1].Position)
	}
	if rides[2].Position != 3 {
		t.Errorf("rides[2].Position = %d, want 3", rides[2].Position)
	}
	if rides[0].Filename != "newest.gpx" {
		t.Errorf("rides[0].Filename = %q, want \"newest.gpx\"", rides[0].Filename)
	}
}
```

- [ ] **Step 2: Run — verify it fails**

```bash
go test ./internal/store/... -run "TestListRides_HasPositions" -v
```

Expected: FAIL — `Position` is 0 on all rides (the query doesn't compute it yet).

- [ ] **Step 3: Update `ListRides` in `internal/store/rides.go`**

Replace the `rows, err := s.db.Query(...)` block inside `ListRides` (the big SELECT at the bottom of the function) and its `scanRide` call in the loop. The full updated function body:

```go
func (s *Store) ListRides(f RideFilters) ([]parser.Ride, int, error) {
	if f.Limit == 0 {
		f.Limit = 10
	}
	if f.Page == 0 {
		f.Page = 1
	}

	where, args := buildRideWhere(f)
	countRow := s.db.QueryRow(`SELECT COUNT(*) FROM rides`+where, args...)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rides: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)
	rows, err := s.db.Query(`
		WITH ranked AS (
			SELECT id, filename, recorded_at, distance_m, duration_s,
				elevation_gain_m, avg_speed_mps, max_speed_mps,
				avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
				calories, source_format,
				ROW_NUMBER() OVER (ORDER BY recorded_at DESC, id DESC) AS position
			FROM rides
		)
		SELECT id, filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format, position
		FROM ranked`+where+` ORDER BY recorded_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rides: %w", err)
	}
	defer rows.Close()

	rides := make([]parser.Ride, 0)
	for rows.Next() {
		r, err := scanRankedRide(rows)
		if err != nil {
			return nil, 0, err
		}
		rides = append(rides, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return rides, total, nil
}
```

- [ ] **Step 4: Run — verify new test passes**

```bash
go test ./internal/store/... -run "TestListRides_HasPositions" -v
```

Expected: PASS.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat: compute global position in ListRides via window function CTE"
```

---

## Task 4: Update `ride <n>` command to resolve by position

**Files:**
- Modify: `cmd/ride.go`

- [ ] **Step 1: Replace `PersistentPreRunE` in `cmd/ride.go`**

The current `PersistentPreRunE` parses `args[0]` as a DB ID and stores it directly. Replace it so it resolves position → DB ID:

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("requires a ride position")
    }
    pos, err := strconv.ParseInt(args[0], 10, 64)
    if err != nil {
        return fmt.Errorf("invalid position %q: must be a number", args[0])
    }
    dbPath, err := store.DefaultPath()
    if err != nil {
        return err
    }
    s, err := store.Open(dbPath)
    if err != nil {
        return err
    }
    defer s.Close()
    ride, err := s.GetRideByPosition(pos)
    if err != nil {
        return err
    }
    currentRideID = ride.ID
    return nil
},
```

The `Use` field can also be updated for clarity:

```go
Use:   "ride <position>",
```

- [ ] **Step 2: Build and manually test**

```bash
go build -o paceline .
./paceline rides          # note the # of an existing ride, e.g. 1
./paceline ride 1         # should show that ride's detail
./paceline ride 999       # should print: "no ride at position 999 — run 'paceline rides' to see available rides"
./paceline ride abc       # should print: invalid position error
```

- [ ] **Step 3: Commit**

```bash
git add cmd/ride.go
git commit -m "feat: ride command resolves by position instead of DB ID"
```

---

## Task 5: Update `delete ride <n>` command to resolve by position

**Files:**
- Modify: `cmd/delete.go`

- [ ] **Step 1: Replace `runDeleteRide` in `cmd/delete.go`**

```go
func runDeleteRide(cmd *cobra.Command, args []string) error {
	pos, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid position %q: must be a number", args[0])
	}

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	ride, err := s.GetRideByPosition(pos)
	if err != nil {
		return err
	}

	distanceKM := ride.DistanceM / 1000.0
	prompt := fmt.Sprintf("Delete ride #%d (%s, %.1f km)?",
		ride.Position, ride.RecordedAt.Format("2006-01-02"), distanceKM)

	if !confirm(prompt, deleteForce) {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := s.DeleteRide(ride.ID); err != nil {
		return err
	}
	fmt.Printf("Deleted ride #%d.\n", ride.Position)
	return nil
}
```

Also update the `Use` field on `deleteRideCmd`:

```go
var deleteRideCmd = &cobra.Command{
	Use:   "ride <position>",
	Short: "Delete a specific ride and its stream data",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteRide,
}
```

- [ ] **Step 2: Build and manually test**

```bash
go build -o paceline .
./paceline rides                   # note position of a ride you can delete
./paceline delete ride <position>  # should prompt with correct date/distance
./paceline delete ride 999         # should print: "no ride at position 999 — run 'paceline rides' to see available rides"
```

- [ ] **Step 3: Commit**

```bash
git add cmd/delete.go
git commit -m "feat: delete ride command resolves by position instead of DB ID"
```

---

## Task 6: Update display — table and TUI

**Files:**
- Modify: `internal/display/table.go`
- Modify: `internal/display/tui.go`
- Modify: `internal/display/display_test.go`

- [ ] **Step 1: Write a failing test for the position column**

Add to `internal/display/display_test.go`. Add `"bytes"` and `"strings"` to the import block:

```go
import (
    "bytes"
    "strings"
    "testing"
    "time"

    "github.com/hollandclarke/paceline/internal/parser"
)
```

Then append the test:

```go
func TestPrintRideList_ShowsPositionColumn(t *testing.T) {
    var buf bytes.Buffer
    rides := []parser.Ride{
        {
            Position:       42,
            RecordedAt:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
            DistanceM:      30000,
            DurationS:      3600,
            ElevationGainM: 500,
            AvgSpeedMPS:    8.3,
        },
    }
    PrintRideList(&buf, rides, 1, 1, 10, false)
    output := buf.String()
    if !strings.Contains(output, "#") {
        t.Errorf("expected '#' column header in output, got:\n%s", output)
    }
    if !strings.Contains(output, "42") {
        t.Errorf("expected position '42' in output, got:\n%s", output)
    }
}
```

- [ ] **Step 2: Run — verify it fails**

```bash
go test ./internal/display/... -run "TestPrintRideList_ShowsPositionColumn" -v
```

Expected: FAIL — output contains `ID` header, not `#`.

- [ ] **Step 3: Update `PrintRideList` in `internal/display/table.go`**

Change the header and the ID column in `PrintRideList`:

```go
table.Header([]string{"#", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
for _, r := range rides {
    table.Append([]string{
        strconv.FormatInt(r.Position, 10),
        r.RecordedAt.Format("2006-01-02"),
        fmt.Sprintf("%.1f km", r.DistanceM/1000),
        formatDuration(r.DurationS),
        fmt.Sprintf("%.0f m", r.ElevationGainM),
        fmt.Sprintf("%.1f km/h", r.AvgSpeedMPS*3.6),
    })
}
```

- [ ] **Step 4: Run — verify test passes**

```bash
go test ./internal/display/... -run "TestPrintRideList_ShowsPositionColumn" -v
```

Expected: PASS.

- [ ] **Step 5: Update the TUI in `internal/display/tui.go`**

In the `View()` method, change the header line and the per-row format line:

```go
// Header line (was "ID", change to "#")
header := fmt.Sprintf("  %-4s  %-12s  %-10s  %-10s  %-10s\n", "#", "Date", "Distance", "Duration", "Elevation")

// Per-row line (was r.ID, change to r.Position)
fmt.Fprintf(&sb, "%s%-4d  %-12s  %-10s  %-10s  %-10s\n",
    cursor,
    r.Position,
    r.RecordedAt.Format("2006-01-02"),
    fmt.Sprintf("%.1f km", r.DistanceM/1000),
    formatDuration(r.DurationS),
    fmt.Sprintf("%.0f m", r.ElevationGainM),
)
```

- [ ] **Step 6: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/display/table.go internal/display/tui.go internal/display/display_test.go
git commit -m "feat: display position as '#' column in rides list and TUI"
```

---

## Task 7: Final verification

- [ ] **Step 1: Full build and test run**

```bash
go build -o paceline . && go test ./... && go vet ./...
```

Expected: binary built, all tests pass, no vet issues.

- [ ] **Step 2: End-to-end smoke test**

```bash
./paceline rides                          # list shows # column with 1, 2, 3...
./paceline ride 1                         # shows most recent ride detail
./paceline ride 1 stream                  # shows stream chart for most recent ride
./paceline rides --json | head -5         # JSON shows "id": 1 (position), no raw DB id
./paceline delete ride 1 --force          # deletes most recent ride
./paceline rides                          # remaining rides renumber from 1
```

- [ ] **Step 3: Commit** (if any last-minute fixes were needed)

```bash
git add -A
git commit -m "chore: final position-based IDs verification"
```
