# Stats All-Time Default Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change `paceline stats` with no flags to show all-time aggregated totals labelled "all time" instead of defaulting to the current month.

**Architecture:** The fix is confined to `cmd/stats.go`. Remove the `noFlags` block that injects the current year/month into `StatsFilters` — `buildStatsWhere` already handles an empty `StatsFilters{}` by producing no `WHERE` clause, so `GetStats` returns all-time data. A store-level test is added first to confirm this behaviour before touching the cmd layer.

**Tech Stack:** Go, Cobra, DuckDB (via `internal/store`)

---

### Task 1: Add store test confirming `GetStats` with empty filters returns all rides

**Files:**
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Add the test**

Open `internal/store/rides_test.go` and append the following function after `TestGetStats_FiltersByYearAndMonth` (around line 222):

```go
func TestGetStats_AllTime(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{
			Filename:       "ride_2023.gpx",
			RecordedAt:     time.Date(2023, 3, 10, 9, 0, 0, 0, time.UTC),
			DistanceM:      20000,
			DurationS:      2400,
			ElevationGainM: 200,
			SourceFormat:   "gpx",
		},
		{
			Filename:       "ride_2025.gpx",
			RecordedAt:     time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
			DistanceM:      30000,
			DurationS:      3600,
			ElevationGainM: 400,
			SourceFormat:   "gpx",
		},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	stats, err := s.GetStats(store.StatsFilters{})
	if err != nil {
		t.Fatalf("GetStats all-time: %v", err)
	}
	if stats.RideCount != 2 {
		t.Errorf("RideCount: got %d, want 2", stats.RideCount)
	}
	if stats.TotalDistanceM != 50000 {
		t.Errorf("TotalDistanceM: got %v, want 50000", stats.TotalDistanceM)
	}
	if stats.TotalDurationS != 6000 {
		t.Errorf("TotalDurationS: got %d, want 6000", stats.TotalDurationS)
	}
	if stats.TotalElevationM != 600 {
		t.Errorf("TotalElevationM: got %v, want 600", stats.TotalElevationM)
	}
}
```

- [ ] **Step 2: Run the test**

```bash
go test ./internal/store/... -run TestGetStats_AllTime -v
```

Expected: `PASS` — the store already handles empty filters correctly; this test locks in that contract.

- [ ] **Step 3: Commit**

```bash
git add internal/store/rides_test.go
git commit -m "test: add TestGetStats_AllTime to confirm empty-filter behaviour"
```

---

### Task 2: Remove the current-month default from `cmd/stats.go`

**Files:**
- Modify: `cmd/stats.go`

- [ ] **Step 1: Remove the `noFlags` injection block and update the label**

In `cmd/stats.go`, the current `runStats` function contains this block (around lines 44–70):

```go
now := time.Now()
f := store.StatsFilters{}

noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0
if noFlags {
    y, m := now.Year(), int(now.Month())
    f.Year = &y
    f.Month = &m
} else {
    if statsYear != 0 {
        f.Year = &statsYear
    }
    if statsMonth != 0 {
        f.Month = &statsMonth
        if f.Year == nil {
            y := now.Year()
            f.Year = &y
        }
    }
    if statsWeek != 0 {
        f.Week = &statsWeek
        if f.Year == nil {
            y := now.Year()
            f.Year = &y
        }
    }
}

// Build human-readable label from the active filters.
var label string
if noFlags {
    label = "current month"
} else {
    var labelParts []string
    if f.Year != nil {
        labelParts = append(labelParts, fmt.Sprintf("%d", *f.Year))
    }
    if f.Month != nil {
        labelParts = append(labelParts, fmt.Sprintf("month %02d", *f.Month))
    }
    if f.Week != nil {
        labelParts = append(labelParts, fmt.Sprintf("week %d", *f.Week))
    }
    label = strings.Join(labelParts, " ")
    if label == "" {
        label = "current month"
    }
}
```

Replace it with:

```go
now := time.Now()
f := store.StatsFilters{}

noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0

if statsYear != 0 {
    f.Year = &statsYear
}
if statsMonth != 0 {
    f.Month = &statsMonth
    if f.Year == nil {
        y := now.Year()
        f.Year = &y
    }
}
if statsWeek != 0 {
    f.Week = &statsWeek
    if f.Year == nil {
        y := now.Year()
        f.Year = &y
    }
}

// Build human-readable label from the active filters.
var label string
if noFlags {
    label = "all time"
} else {
    var labelParts []string
    if f.Year != nil {
        labelParts = append(labelParts, fmt.Sprintf("%d", *f.Year))
    }
    if f.Month != nil {
        labelParts = append(labelParts, fmt.Sprintf("month %02d", *f.Month))
    }
    if f.Week != nil {
        labelParts = append(labelParts, fmt.Sprintf("week %d", *f.Week))
    }
    label = strings.Join(labelParts, " ")
    if label == "" {
        label = "all time"
    }
}
```

Key changes:
- The `if noFlags { ... } else { ... }` wrapper around filter assignment is gone — flags are applied unconditionally (they're guarded by `!= 0` checks anyway).
- `label = "current month"` → `label = "all time"` in the `noFlags` branch.
- The fallback `label = "current month"` (unreachable dead code) → `label = "all time"` for consistency.

- [ ] **Step 2: Build to confirm no compile errors**

```bash
go build -o paceline .
```

Expected: exits 0, no errors.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 4: Smoke-test the output**

```bash
./paceline stats
```

Expected: output begins with `Stats: all time` followed by the totals table.

```bash
./paceline stats --month=5
```

Expected: output begins with `Stats: YYYY month 05` (current year injected).

```bash
./paceline stats --year=2024
```

Expected: output begins with `Stats: 2024`.

- [ ] **Step 5: Commit**

```bash
git add cmd/stats.go
git commit -m "feat: change paceline stats default to all-time totals

Remove the implicit current-month filter applied when no flags are
given. Empty StatsFilters{} already produces no WHERE clause in the
store layer, so GetStats now returns all-time aggregated data.
Label updated from 'current month' to 'all time'."
```

---

### Task 3: Clean up binary and verify

- [ ] **Step 1: Remove the dev binary**

```bash
rm -f paceline
```

- [ ] **Step 2: Run the full test suite one final time**

```bash
go test ./...
```

Expected: all pass, no failures.
