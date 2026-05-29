# Date Range Filtering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--from` and `--to` (`YYYY-MM-DD`) flags to the `rides`, `stats`, and `records` commands so users can filter by an arbitrary date range.

**Architecture:** Add `From *time.Time` and `To *time.Time` to the three filter structs in `internal/store/rides.go`, extract a shared `appendDateRangeClauses` helper that appends the two SQL clauses, and wire up new flags in each of the three command files with input validation. No schema changes; no display changes.

**Tech Stack:** Go standard library (`time`), DuckDB SQL (`::DATE` cast, `INTERVAL 1 DAY`), Cobra flags.

---

## File Map

| File | Change |
|---|---|
| `internal/store/rides.go` | Add `From`/`To` to `RideFilters`, `StatsFilters`, `RecordsFilters`; add `appendDateRangeClauses`; call it from all three `buildXxxWhere` functions |
| `internal/store/rides_test.go` | New test functions for `From`/`To` on all three filter types |
| `cmd/rides.go` | Add `--from`/`--to` flags; parse + validate; populate `RideFilters` |
| `cmd/stats.go` | Add `--from`/`--to` flags; parse + validate; populate `StatsFilters`; extend label |
| `cmd/records.go` | Add `--from`/`--to` flags; parse + validate; populate `RecordsFilters`; extend label |

---

## Task 1: Store layer — RideFilters (TDD)

**Files:**
- Modify: `internal/store/rides.go`
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Write failing tests for ListRides with From/To**

Append to `internal/store/rides_test.go`:

```go
func TestListRides_FromFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{From: &from, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
	for _, r := range result {
		if r.RecordedAt.Before(from) {
			t.Errorf("ride %q has date %v before --from %v", r.Filename, r.RecordedAt, from)
		}
	}
}

func TestListRides_ToFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
}

func TestListRides_FromToFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "dec.gpx", RecordedAt: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{From: &from, To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2 (mar+jun)", total)
	}
	if len(result) != 2 {
		t.Errorf("len: got %d, want 2", len(result))
	}
}

func TestListRides_ToFilter_Inclusive(t *testing.T) {
	s := openTestStore(t)

	// A ride recorded exactly on the --to date must be included.
	boundary := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	rides := []parser.Ride{
		{Filename: "on_boundary.gpx", RecordedAt: boundary, SourceFormat: "gpx"},
		{Filename: "after_boundary.gpx", RecordedAt: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	to := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	result, total, err := s.ListRides(store.RideFilters{To: &to, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if total != 1 {
		t.Errorf("total: got %d, want 1 (boundary date must be included)", total)
	}
	if len(result) == 0 || result[0].Filename != "on_boundary.gpx" {
		t.Errorf("expected on_boundary.gpx to be included")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/... -run "TestListRides_FromFilter|TestListRides_ToFilter|TestListRides_FromToFilter|TestListRides_ToFilter_Inclusive" -v
```

Expected: compilation failure — `store.RideFilters` has no field `From` or `To`.

- [ ] **Step 3: Add From/To to RideFilters, implement helper, update buildRideWhere**

In `internal/store/rides.go`, update `RideFilters`:

```go
type RideFilters struct {
	Year  *int
	Month *int
	Date  *time.Time
	From  *time.Time
	To    *time.Time
	Page  int
	Limit int
}
```

Add the helper function (place it just above `buildRideWhere`):

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

Update `buildRideWhere` to call the helper at the end (after the existing `f.Date` block):

```go
func buildRideWhere(f RideFilters) (string, []any) { //nolint:gocritic // unnamedResult: intentional, named returns add noise here
	var clauses []string
	var args []any

	if f.Year != nil {
		clauses = append(clauses, "EXTRACT(YEAR FROM recorded_at) = ?")
		args = append(args, *f.Year)
	}
	if f.Month != nil {
		clauses = append(clauses, "EXTRACT(MONTH FROM recorded_at) = ?")
		args = append(args, *f.Month)
	}
	if f.Date != nil {
		clauses = append(clauses, "DATE_TRUNC('day', recorded_at) = DATE_TRUNC('day', ?::TIMESTAMP)")
		args = append(args, f.Date.Format(time.RFC3339))
	}
	clauses, args = appendDateRangeClauses(clauses, args, f.From, f.To)

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/store/... -run "TestListRides_FromFilter|TestListRides_ToFilter|TestListRides_FromToFilter|TestListRides_ToFilter_Inclusive" -v
```

Expected: all four tests PASS.

- [ ] **Step 5: Run full store test suite to check for regressions**

```bash
go test ./internal/store/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat(store): add From/To range filter to RideFilters"
```

---

## Task 2: Store layer — StatsFilters and RecordsFilters (TDD)

**Files:**
- Modify: `internal/store/rides.go`
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Write failing tests for GetStats and GetRecords with From/To**

Append to `internal/store/rides_test.go`:

```go
func TestGetStats_FromToFilter(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), DistanceM: 10000, DurationS: 1200, ElevationGainM: 100, SourceFormat: "gpx"},
		{Filename: "mar.gpx", RecordedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), DistanceM: 20000, DurationS: 2400, ElevationGainM: 200, SourceFormat: "gpx"},
		{Filename: "dec.gpx", RecordedAt: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), DistanceM: 30000, DurationS: 3600, ElevationGainM: 300, SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	stats, err := s.GetStats(store.StatsFilters{From: &from, To: &to})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.RideCount != 1 {
		t.Errorf("RideCount: got %d, want 1 (only mar)", stats.RideCount)
	}
	if stats.TotalDistanceM != 20000 {
		t.Errorf("TotalDistanceM: got %v, want 20000", stats.TotalDistanceM)
	}
}

func TestGetStats_FromOnly(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), DistanceM: 10000, DurationS: 1200, ElevationGainM: 100, SourceFormat: "gpx"},
		{Filename: "jun.gpx", RecordedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), DistanceM: 20000, DurationS: 2400, ElevationGainM: 200, SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	stats, err := s.GetStats(store.StatsFilters{From: &from})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.RideCount != 1 {
		t.Errorf("RideCount: got %d, want 1", stats.RideCount)
	}
	if stats.TotalDistanceM != 20000 {
		t.Errorf("TotalDistanceM: got %v, want 20000", stats.TotalDistanceM)
	}
}

func TestGetRecords_FromToFilter(t *testing.T) {
	s := openTestStore(t)

	// Insert two rides: only the one inside the range should contribute to records.
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "outside.gpx",
		RecordedAt:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		DistanceM:    200000,
		DurationS:    7200,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert outside: %v", err)
	}
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "inside.gpx",
		RecordedAt:   time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		DistanceM:    50000,
		DurationS:    3600,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert inside: %v", err)
	}

	from := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	recs, err := s.GetRecords(store.RecordsFilters{From: &from, To: &to})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if recs.LongestDistanceM == nil {
		t.Fatal("expected LongestDistanceM to be set")
	}
	// Must be the inside ride (50000), not outside (200000).
	if recs.LongestDistanceM.RawValue != 50000 {
		t.Errorf("LongestDistanceM: got %v, want 50000 (only inside-range ride)", recs.LongestDistanceM.RawValue)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/... -run "TestGetStats_FromToFilter|TestGetStats_FromOnly|TestGetRecords_FromToFilter" -v
```

Expected: compilation failure — `StatsFilters` and `RecordsFilters` have no `From`/`To` fields.

- [ ] **Step 3: Add From/To to StatsFilters and RecordsFilters, update buildStatsWhere and buildRecordsWhere**

In `internal/store/rides.go`, update `StatsFilters`:

```go
type StatsFilters struct {
	Year  *int
	Month *int
	Week  *int
	From  *time.Time
	To    *time.Time
}
```

Update `RecordsFilters`:

```go
type RecordsFilters struct {
	Year  *int
	Month *int
	Week  *int
	From  *time.Time
	To    *time.Time
}
```

Update `buildStatsWhere` to call the helper at the end (after the existing `f.Week` block):

```go
func buildStatsWhere(f StatsFilters) (string, []any) { //nolint:gocritic // unnamedResult: intentional, named returns add noise here
	var clauses []string
	var args []any

	if f.Year != nil {
		clauses = append(clauses, "EXTRACT(YEAR FROM recorded_at) = ?")
		args = append(args, *f.Year)
	}
	if f.Month != nil {
		clauses = append(clauses, "EXTRACT(MONTH FROM recorded_at) = ?")
		args = append(args, *f.Month)
	}
	if f.Week != nil {
		clauses = append(clauses, "EXTRACT(WEEK FROM recorded_at) = ?")
		args = append(args, *f.Week)
	}
	clauses, args = appendDateRangeClauses(clauses, args, f.From, f.To)

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
```

Update `buildRecordsWhere` to call the helper at the end (after the existing `f.Week` block):

```go
func buildRecordsWhere(f RecordsFilters) (string, []any) { //nolint:gocritic // unnamedResult: intentional, named returns add noise here
	var clauses []string
	var args []any

	if f.Year != nil {
		clauses = append(clauses, "EXTRACT(YEAR FROM recorded_at) = ?")
		args = append(args, *f.Year)
	}
	if f.Month != nil {
		clauses = append(clauses, "EXTRACT(MONTH FROM recorded_at) = ?")
		args = append(args, *f.Month)
	}
	if f.Week != nil {
		clauses = append(clauses, "EXTRACT(WEEK FROM recorded_at) = ?")
		args = append(args, *f.Week)
	}
	clauses, args = appendDateRangeClauses(clauses, args, f.From, f.To)

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
```

- [ ] **Step 4: Run new tests to verify they pass**

```bash
go test ./internal/store/... -run "TestGetStats_FromToFilter|TestGetStats_FromOnly|TestGetRecords_FromToFilter" -v
```

Expected: all three tests PASS.

- [ ] **Step 5: Run full store test suite to check for regressions**

```bash
go test ./internal/store/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat(store): add From/To range filter to StatsFilters and RecordsFilters"
```

---

## Task 3: rides command — --from/--to flags

**Files:**
- Modify: `cmd/rides.go`

- [ ] **Step 1: Add flag variables, register flags, and wire into RideFilters**

Replace the `var` block and `init()` and `runRides` in `cmd/rides.go` with the following (full file shown):

```go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var ridesCmd = &cobra.Command{
	Use:   "rides",
	Short: "List rides",
	RunE:  runRides,
}

var (
	ridesYear  int
	ridesMonth int
	ridesDate  string
	ridesFrom  string
	ridesTo    string
	ridesPage  int
	ridesLimit int
)

func init() {
	rootCmd.AddCommand(ridesCmd)
	ridesCmd.Flags().IntVar(&ridesYear, "year", 0, "filter by year (e.g. 2024)")
	ridesCmd.Flags().IntVar(&ridesMonth, "month", 0, "filter by month (1-12)")
	ridesCmd.Flags().StringVar(&ridesDate, "date", "", "filter by date (YYYY-MM-DD)")
	ridesCmd.Flags().StringVar(&ridesFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	ridesCmd.Flags().StringVar(&ridesTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
	ridesCmd.Flags().IntVar(&ridesPage, "page", 1, "page number")
	ridesCmd.Flags().IntVar(&ridesLimit, "limit", 10, "results per page")
}

func runRides(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	f := store.RideFilters{Page: ridesPage, Limit: ridesLimit}

	if ridesYear != 0 {
		f.Year = &ridesYear
	}
	if ridesMonth != 0 {
		if f.Year == nil {
			y := time.Now().Year()
			f.Year = &y
		}
		f.Month = &ridesMonth
	}
	if ridesDate != "" {
		t, err := time.Parse("2006-01-02", ridesDate)
		if err != nil {
			return fmt.Errorf("invalid --date %q: use YYYY-MM-DD", ridesDate)
		}
		f.Date = &t
	}
	if ridesFrom != "" {
		t, err := time.Parse("2006-01-02", ridesFrom)
		if err != nil {
			return fmt.Errorf("invalid --from %q: use YYYY-MM-DD", ridesFrom)
		}
		f.From = &t
	}
	if ridesTo != "" {
		t, err := time.Parse("2006-01-02", ridesTo)
		if err != nil {
			return fmt.Errorf("invalid --to %q: use YYYY-MM-DD", ridesTo)
		}
		f.To = &t
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return fmt.Errorf("--from must not be after --to")
	}

	loadPage := func(page int) ([]parser.Ride, int, error) {
		f2 := f
		f2.Page = page
		return s.ListRides(f2)
	}

	rides, total, err := s.ListRides(f)
	if err != nil {
		return err
	}

	if display.IsTTY() && !jsonOutput {
		selected, err := display.RunRidesTUI(os.Stdout, rides, total, ridesLimit, loadPage, cfg.Units)
		if err != nil {
			return err
		}
		if selected != nil {
			display.PrintRideDetail(os.Stdout, *selected, false, cfg.Units)
		}
		return nil
	}

	display.PrintRideList(os.Stdout, rides, total, ridesPage, ridesLimit, jsonOutput, cfg.Units)
	return nil
}
```

- [ ] **Step 2: Build and verify no compilation errors**

```bash
make build
```

Expected: compiles cleanly, binary at `./paceline`.

- [ ] **Step 3: Run full test suite**

```bash
make all
```

Expected: vet + tests + lint all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/rides.go
git commit -m "feat(cmd): add --from/--to flags to rides command"
```

---

## Task 4: stats command — --from/--to flags

**Files:**
- Modify: `cmd/stats.go`

- [ ] **Step 1: Add flag variables, register flags, wire into StatsFilters, extend label**

Replace the contents of `cmd/stats.go` with:

```go
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregated ride totals",
	RunE:  runStats,
}

var (
	statsYear  int
	statsMonth int
	statsWeek  int
	statsFrom  string
	statsTo    string
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsYear, "year", 0, "filter by year (e.g. 2024)")
	statsCmd.Flags().IntVar(&statsMonth, "month", 0, "filter by month (1-12)")
	statsCmd.Flags().IntVar(&statsWeek, "week", 0, "filter by ISO week number (1-53)")
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
}

func runStats(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	f := store.StatsFilters{}

	noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0 && statsFrom == "" && statsTo == ""

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
	if statsFrom != "" {
		t, err := time.Parse("2006-01-02", statsFrom)
		if err != nil {
			return fmt.Errorf("invalid --from %q: use YYYY-MM-DD", statsFrom)
		}
		f.From = &t
	}
	if statsTo != "" {
		t, err := time.Parse("2006-01-02", statsTo)
		if err != nil {
			return fmt.Errorf("invalid --to %q: use YYYY-MM-DD", statsTo)
		}
		f.To = &t
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return fmt.Errorf("--from must not be after --to")
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
		if f.From != nil && f.To != nil {
			labelParts = append(labelParts, fmt.Sprintf("%s to %s", f.From.Format("2006-01-02"), f.To.Format("2006-01-02")))
		} else if f.From != nil {
			labelParts = append(labelParts, fmt.Sprintf("from %s", f.From.Format("2006-01-02")))
		} else if f.To != nil {
			labelParts = append(labelParts, fmt.Sprintf("to %s", f.To.Format("2006-01-02")))
		}
		label = strings.Join(labelParts, " ")
		if label == "" {
			label = "all time"
		}
	}

	st, err := s.GetStats(f)
	if err != nil {
		return err
	}

	display.PrintStats(os.Stdout, st, label, jsonOutput, cfg.Units)
	return nil
}
```

- [ ] **Step 2: Build and verify no compilation errors**

```bash
make build
```

Expected: compiles cleanly.

- [ ] **Step 3: Run full test suite**

```bash
make all
```

Expected: vet + tests + lint all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/stats.go
git commit -m "feat(cmd): add --from/--to flags to stats command"
```

---

## Task 5: records command — --from/--to flags

**Files:**
- Modify: `cmd/records.go`

- [ ] **Step 1: Add flag variables, register flags, wire into RecordsFilters, extend label**

Replace the contents of `cmd/records.go` with:

```go
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Show personal records (all-time bests)",
	RunE:  runRecords,
}

var (
	recordsYear  int
	recordsMonth int
	recordsWeek  int
	recordsFrom  string
	recordsTo    string
)

func init() {
	rootCmd.AddCommand(recordsCmd)
	recordsCmd.Flags().IntVar(&recordsYear, "year", 0, "filter by year (e.g. 2024)")
	recordsCmd.Flags().IntVar(&recordsMonth, "month", 0, "filter by month (1-12)")
	recordsCmd.Flags().IntVar(&recordsWeek, "week", 0, "filter by ISO week number (1-53)")
	recordsCmd.Flags().StringVar(&recordsFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	recordsCmd.Flags().StringVar(&recordsTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
}

func runRecords(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	f := store.RecordsFilters{}

	noFlags := recordsYear == 0 && recordsMonth == 0 && recordsWeek == 0 && recordsFrom == "" && recordsTo == ""

	if recordsYear != 0 {
		f.Year = &recordsYear
	}
	if recordsMonth != 0 {
		f.Month = &recordsMonth
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}
	if recordsWeek != 0 {
		f.Week = &recordsWeek
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}
	if recordsFrom != "" {
		t, err := time.Parse("2006-01-02", recordsFrom)
		if err != nil {
			return fmt.Errorf("invalid --from %q: use YYYY-MM-DD", recordsFrom)
		}
		f.From = &t
	}
	if recordsTo != "" {
		t, err := time.Parse("2006-01-02", recordsTo)
		if err != nil {
			return fmt.Errorf("invalid --to %q: use YYYY-MM-DD", recordsTo)
		}
		f.To = &t
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return fmt.Errorf("--from must not be after --to")
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
		if f.From != nil && f.To != nil {
			labelParts = append(labelParts, fmt.Sprintf("%s to %s", f.From.Format("2006-01-02"), f.To.Format("2006-01-02")))
		} else if f.From != nil {
			labelParts = append(labelParts, fmt.Sprintf("from %s", f.From.Format("2006-01-02")))
		} else if f.To != nil {
			labelParts = append(labelParts, fmt.Sprintf("to %s", f.To.Format("2006-01-02")))
		}
		label = strings.Join(labelParts, " ")
		if label == "" {
			label = "all time"
		}
	}

	recs, err := s.GetRecords(f)
	if err != nil {
		return err
	}

	display.PrintRecords(os.Stdout, recs, label, jsonOutput, cfg.Units)
	return nil
}
```

- [ ] **Step 2: Build and verify no compilation errors**

```bash
make build
```

Expected: compiles cleanly.

- [ ] **Step 3: Run full test suite**

```bash
make all
```

Expected: vet + tests + lint all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/records.go
git commit -m "feat(cmd): add --from/--to flags to records command"
```

---

## Task 6: Update docs

**Files:**
- Modify: `docs/feature-roadmap.md`
- Modify: `README.md`

- [ ] **Step 1: Mark date-range filtering as completed in the roadmap**

In `docs/feature-roadmap.md`, update the Priority Matrix row for item #3:

```markdown
| 3 | [Date-range filtering](#3-date-range-filtering) | 🔴 High | Low | ✅ Completed |
```

In the `### 3. Date-Range Filtering` section body, add a completed note and remove the shorthand keywords from the proposed commands (they are deferred). The shorthand mention stays as a future note. Add above the `---` divider at the end of that section:

```markdown
> **Status: Completed.** `--from` and `--to` flags are live on `rides`, `stats`, and `records`. Shorthand keywords (`--range last-30-days`, `--range ytd`, etc.) remain a potential follow-on feature.
```

- [ ] **Step 2: Update README `rides` command flags table**

In `README.md`, under the `### \`paceline rides\`` section, add two rows to the **Flags** table:

```markdown
| `--from` | — | Filter rides on or after this date `YYYY-MM-DD` |
| `--to` | — | Filter rides on or before this date `YYYY-MM-DD` |
```

Also add example usage lines in the code block under that section:

```bash
paceline rides --from 2025-01-01 --to 2025-03-31
paceline rides --from 2025-06-01               # on or after
paceline rides --to 2025-06-30                 # on or before
```

- [ ] **Step 3: Update README `stats` command section**

In `README.md`, under `### \`paceline stats\``, add two example lines to the code block:

```bash
paceline stats --from 2025-01-01 --to 2025-03-31
paceline stats --from 2025-01-01               # open-ended range
```

- [ ] **Step 4: Update README `records` command section**

In `README.md`, under `### \`paceline records\``, add two example lines to the code block:

```bash
paceline records --from 2025-01-01 --to 2025-03-31
paceline records --from 2025-06-01
```

- [ ] **Step 5: Commit**

```bash
git add docs/feature-roadmap.md README.md
git commit -m "docs: mark date-range filtering complete, update README"
```
