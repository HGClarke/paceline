# Personal Records Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `paceline records` command that queries the database for 9 personal best categories and renders them in a 3-column table (Record | Value | Date), with optional year/month/week filtering.

**Architecture:** Approach A — 9 sequential queries (8 on the `rides` table via a shared helper, 1 joining `streams` to `rides` for peak altitude). New types `RecordsFilters`, `PersonalRecord`, and `Records` live in the store layer. `PrintRecords` in the display layer renders the table and omits rows for nil records. No schema changes.

**Tech Stack:** Go, DuckDB (`go-duckdb`), `tablewriter`, `cobra`

---

## File Map

| File | Change |
|---|---|
| `internal/store/rides.go` | Add `RecordsFilters`, `PersonalRecord`, `Records`, `GetRecords`, `buildRecordsWhere`, `queryMaxRidesRecord`, `queryMaxAltitudeRecord` |
| `internal/store/rides_test.go` | Add 5 store tests |
| `internal/display/table.go` | Add `PrintRecords` |
| `internal/display/display_test.go` | Add 5 display tests |
| `cmd/records.go` | New file — `recordsCmd` |

---

## Task 1: Store types, query helpers, and `GetRecords`

**Files:**
- Modify: `internal/store/rides.go`
- Test: `internal/store/rides_test.go`

- [ ] **Step 1: Write the failing store tests**

Append to `internal/store/rides_test.go`:

```go
func TestGetRecords_Empty(t *testing.T) {
	s := openTestStore(t)
	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if recs.LongestDistanceM != nil {
		t.Error("expected nil LongestDistanceM")
	}
	if recs.LongestDurationS != nil {
		t.Error("expected nil LongestDurationS")
	}
	if recs.MostElevationGainM != nil {
		t.Error("expected nil MostElevationGainM")
	}
	if recs.HighestAvgPowerW != nil {
		t.Error("expected nil HighestAvgPowerW")
	}
	if recs.HighestAvgSpeedMPS != nil {
		t.Error("expected nil HighestAvgSpeedMPS")
	}
	if recs.HighestAvgHRBPM != nil {
		t.Error("expected nil HighestAvgHRBPM")
	}
	if recs.HighestMaxSpeedMPS != nil {
		t.Error("expected nil HighestMaxSpeedMPS")
	}
	if recs.MostCaloriesKcal != nil {
		t.Error("expected nil MostCaloriesKcal")
	}
	if recs.HighestAltitudeM != nil {
		t.Error("expected nil HighestAltitudeM")
	}
}

func TestGetRecords_AllFields(t *testing.T) {
	s := openTestStore(t)

	hr, power, cal := 155, 250, 800
	alt := 1200.0
	rideDate := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	id, err := s.InsertRide(parser.Ride{
		Filename:       "full.gpx",
		RecordedAt:     rideDate,
		DistanceM:      50000,
		DurationS:      7200,
		ElevationGainM: 1500,
		AvgSpeedMPS:    10.0,
		MaxSpeedMPS:    20.0,
		AvgHRBPM:       &hr,
		MaxHRBPM:       &hr,
		AvgPowerW:      &power,
		MaxPowerW:      &power,
		Calories:       &cal,
		SourceFormat:   "gpx",
	})
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}
	if err := s.InsertStreams([]parser.Stream{
		{RideID: id, Timestamp: rideDate, ElapsedS: 0, AltitudeM: &alt},
	}); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	check := func(name string, pr *store.PersonalRecord, wantVal float64) {
		t.Helper()
		if pr == nil {
			t.Errorf("%s: got nil, want %v", name, wantVal)
			return
		}
		if pr.RawValue != wantVal {
			t.Errorf("%s.RawValue: got %v, want %v", name, pr.RawValue, wantVal)
		}
		if pr.Date.Format("2006-01-02") != "2024-06-15" {
			t.Errorf("%s.Date: got %v, want 2024-06-15", name, pr.Date)
		}
	}

	check("LongestDistanceM", recs.LongestDistanceM, 50000)
	check("LongestDurationS", recs.LongestDurationS, 7200)
	check("MostElevationGainM", recs.MostElevationGainM, 1500)
	check("HighestAvgPowerW", recs.HighestAvgPowerW, 250)
	check("HighestAvgSpeedMPS", recs.HighestAvgSpeedMPS, 10.0)
	check("HighestAvgHRBPM", recs.HighestAvgHRBPM, 155)
	check("HighestMaxSpeedMPS", recs.HighestMaxSpeedMPS, 20.0)
	check("MostCaloriesKcal", recs.MostCaloriesKcal, 800)
	check("HighestAltitudeM", recs.HighestAltitudeM, 1200)
}

func TestGetRecords_MissingNullable(t *testing.T) {
	s := openTestStore(t)

	// Insert a ride with no nullable fields and no stream data.
	_, err := s.InsertRide(parser.Ride{
		Filename:       "basic.gpx",
		RecordedAt:     time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		DistanceM:      30000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    8.0,
		MaxSpeedMPS:    15.0,
		SourceFormat:   "gpx",
	})
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	// Non-nullable fields should be populated.
	if recs.LongestDistanceM == nil {
		t.Error("expected LongestDistanceM to be set")
	}
	if recs.LongestDurationS == nil {
		t.Error("expected LongestDurationS to be set")
	}
	if recs.MostElevationGainM == nil {
		t.Error("expected MostElevationGainM to be set")
	}
	if recs.HighestAvgSpeedMPS == nil {
		t.Error("expected HighestAvgSpeedMPS to be set")
	}
	if recs.HighestMaxSpeedMPS == nil {
		t.Error("expected HighestMaxSpeedMPS to be set")
	}

	// Nullable fields should be nil.
	if recs.HighestAvgPowerW != nil {
		t.Errorf("expected nil HighestAvgPowerW, got %+v", recs.HighestAvgPowerW)
	}
	if recs.HighestAvgHRBPM != nil {
		t.Errorf("expected nil HighestAvgHRBPM, got %+v", recs.HighestAvgHRBPM)
	}
	if recs.MostCaloriesKcal != nil {
		t.Errorf("expected nil MostCaloriesKcal, got %+v", recs.MostCaloriesKcal)
	}
	if recs.HighestAltitudeM != nil {
		t.Errorf("expected nil HighestAltitudeM, got %+v", recs.HighestAltitudeM)
	}
}

func TestGetRecords_YearFilter(t *testing.T) {
	s := openTestStore(t)

	// 2024 ride: farther
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "2024.gpx",
		RecordedAt:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		DistanceM:    100000,
		DurationS:    3600,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert 2024: %v", err)
	}
	// 2025 ride: shorter
	if _, err := s.InsertRide(parser.Ride{
		Filename:     "2025.gpx",
		RecordedAt:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		DistanceM:    30000,
		DurationS:    1800,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("insert 2025: %v", err)
	}

	year := 2025
	recs, err := s.GetRecords(store.RecordsFilters{Year: &year})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if recs.LongestDistanceM == nil {
		t.Fatal("expected LongestDistanceM to be set")
	}
	if recs.LongestDistanceM.RawValue != 30000 {
		t.Errorf("LongestDistanceM: got %v, want 30000", recs.LongestDistanceM.RawValue)
	}
	if recs.LongestDistanceM.Date.Year() != 2025 {
		t.Errorf("LongestDistanceM.Date.Year: got %d, want 2025", recs.LongestDistanceM.Date.Year())
	}
}

func TestGetRecords_PicksMax(t *testing.T) {
	s := openTestStore(t)

	dateA := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	dateB := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)

	// Ride A: longer distance, slower speed.
	if _, err := s.InsertRide(parser.Ride{
		Filename:       "ride_a.gpx",
		RecordedAt:     dateA,
		DistanceM:      100000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    5.0,
		MaxSpeedMPS:    10.0,
		SourceFormat:   "gpx",
	}); err != nil {
		t.Fatalf("insert ride A: %v", err)
	}
	// Ride B: shorter distance, faster speed.
	if _, err := s.InsertRide(parser.Ride{
		Filename:       "ride_b.gpx",
		RecordedAt:     dateB,
		DistanceM:      50000,
		DurationS:      7200,
		ElevationGainM: 200,
		AvgSpeedMPS:    20.0,
		MaxSpeedMPS:    40.0,
		SourceFormat:   "gpx",
	}); err != nil {
		t.Fatalf("insert ride B: %v", err)
	}

	recs, err := s.GetRecords(store.RecordsFilters{})
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	// Distance record must come from ride A.
	if recs.LongestDistanceM == nil {
		t.Fatal("expected LongestDistanceM to be set")
	}
	if recs.LongestDistanceM.RawValue != 100000 {
		t.Errorf("LongestDistanceM: got %v, want 100000", recs.LongestDistanceM.RawValue)
	}
	if !recs.LongestDistanceM.Date.Equal(dateA) {
		t.Errorf("LongestDistanceM.Date: got %v, want %v", recs.LongestDistanceM.Date, dateA)
	}

	// Speed record must come from ride B.
	if recs.HighestAvgSpeedMPS == nil {
		t.Fatal("expected HighestAvgSpeedMPS to be set")
	}
	if recs.HighestAvgSpeedMPS.RawValue != 20.0 {
		t.Errorf("HighestAvgSpeedMPS: got %v, want 20.0", recs.HighestAvgSpeedMPS.RawValue)
	}
	if !recs.HighestAvgSpeedMPS.Date.Equal(dateB) {
		t.Errorf("HighestAvgSpeedMPS.Date: got %v, want %v", recs.HighestAvgSpeedMPS.Date, dateB)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/... -run "TestGetRecords" -v
```

Expected: FAIL — `store.RecordsFilters`, `store.PersonalRecord`, `store.Records`, and `store.GetRecords` are undefined.

- [ ] **Step 3: Implement store types, helpers, and `GetRecords`**

Append to the bottom of `internal/store/rides.go`:

```go
// RecordsFilters controls which rides are considered when computing personal records.
type RecordsFilters struct {
	Year  *int
	Month *int
	Week  *int
}

// PersonalRecord holds the raw value and date of a single personal best.
// A nil *PersonalRecord means no data exists for that category.
type PersonalRecord struct {
	RawValue float64   `json:"value"`
	Date     time.Time `json:"date"`
}

// Records holds all personal best categories. Nil fields mean no data.
type Records struct {
	LongestDistanceM   *PersonalRecord `json:"longest_distance_m"`
	LongestDurationS   *PersonalRecord `json:"longest_duration_s"`
	MostElevationGainM *PersonalRecord `json:"most_elevation_gain_m"`
	HighestAvgPowerW   *PersonalRecord `json:"highest_avg_power_w"`
	HighestAvgSpeedMPS *PersonalRecord `json:"highest_avg_speed_mps"`
	HighestAvgHRBPM    *PersonalRecord `json:"highest_avg_hr_bpm"`
	HighestMaxSpeedMPS *PersonalRecord `json:"highest_max_speed_mps"`
	MostCaloriesKcal   *PersonalRecord `json:"most_calories_kcal"`
	HighestAltitudeM   *PersonalRecord `json:"highest_altitude_m"`
}

// GetRecords computes personal bests across all 9 categories using the given filters.
// Records for which no data exists are nil in the returned struct.
func (s *Store) GetRecords(f RecordsFilters) (Records, error) {
	where, args := buildRecordsWhere(f)
	var recs Records
	var err error

	if recs.LongestDistanceM, err = s.queryMaxRidesRecord("distance_m", where, args); err != nil {
		return recs, err
	}
	if recs.LongestDurationS, err = s.queryMaxRidesRecord("duration_s", where, args); err != nil {
		return recs, err
	}
	if recs.MostElevationGainM, err = s.queryMaxRidesRecord("elevation_gain_m", where, args); err != nil {
		return recs, err
	}
	if recs.HighestAvgPowerW, err = s.queryMaxRidesRecord("avg_power_w", where, args); err != nil {
		return recs, err
	}
	if recs.HighestAvgSpeedMPS, err = s.queryMaxRidesRecord("avg_speed_mps", where, args); err != nil {
		return recs, err
	}
	if recs.HighestAvgHRBPM, err = s.queryMaxRidesRecord("avg_hr_bpm", where, args); err != nil {
		return recs, err
	}
	if recs.HighestMaxSpeedMPS, err = s.queryMaxRidesRecord("max_speed_mps", where, args); err != nil {
		return recs, err
	}
	if recs.MostCaloriesKcal, err = s.queryMaxRidesRecord("calories", where, args); err != nil {
		return recs, err
	}
	if recs.HighestAltitudeM, err = s.queryMaxAltitudeRecord(where, args); err != nil {
		return recs, err
	}
	return recs, nil
}

// buildRecordsWhere builds a WHERE clause for RecordsFilters against the rides table.
// The column name "recorded_at" is unqualified and refers to rides.recorded_at.
func buildRecordsWhere(f RecordsFilters) (string, []any) {
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

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// queryMaxRidesRecord returns the PersonalRecord for the given column in the rides table.
// Uses NULLS LAST so that NULL values (e.g. avg_power_w for rides without a power meter)
// sort after non-NULL values; if all values for the column are NULL, returns nil.
// The where string and args come from buildRecordsWhere.
func (s *Store) queryMaxRidesRecord(field, where string, args []any) (*PersonalRecord, error) {
	q := fmt.Sprintf(
		`SELECT CAST(%s AS DOUBLE), recorded_at FROM rides%s ORDER BY %s DESC NULLS LAST LIMIT 1`,
		field, where, field,
	)
	row := s.db.QueryRow(q, args...)
	var val sql.NullFloat64
	var date sql.NullTime
	if err := row.Scan(&val, &date); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query max %s: %w", field, err)
	}
	if !val.Valid {
		return nil, nil
	}
	return &PersonalRecord{RawValue: val.Float64, Date: date.Time}, nil
}

// queryMaxAltitudeRecord returns the PersonalRecord for the highest altitude point
// recorded across all stream data. It joins streams to rides to obtain the ride date.
// The where string and args come from buildRecordsWhere; "recorded_at" is unambiguous
// in the JOIN because the streams table uses "timestamp" for its time column.
func (s *Store) queryMaxAltitudeRecord(where string, args []any) (*PersonalRecord, error) {
	q := `SELECT MAX(s.altitude_m), r.recorded_at
		FROM streams s
		JOIN rides r ON r.id = s.ride_id` + where + `
		GROUP BY r.id, r.recorded_at
		ORDER BY MAX(s.altitude_m) DESC NULLS LAST
		LIMIT 1`
	row := s.db.QueryRow(q, args...)
	var val sql.NullFloat64
	var date sql.NullTime
	if err := row.Scan(&val, &date); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query max altitude: %w", err)
	}
	if !val.Valid {
		return nil, nil
	}
	return &PersonalRecord{RawValue: val.Float64, Date: date.Time}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/store/... -run "TestGetRecords" -v
```

Expected: all 5 `TestGetRecords_*` tests PASS.

- [ ] **Step 5: Run full test suite to check for regressions**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat: add GetRecords to store layer for personal records"
```

---

## Task 2: Display layer — `PrintRecords`

**Files:**
- Modify: `internal/display/table.go`
- Test: `internal/display/display_test.go`

- [ ] **Step 1: Write the failing display tests**

Add the following import to `internal/display/display_test.go` — the file already imports `bytes`, `strings`, `testing`, `time`, and `parser`; add `store` to the import block:

```go
import (
    "bytes"
    "strings"
    "testing"
    "time"

    "github.com/HGClarke/paceline/internal/parser"
    "github.com/HGClarke/paceline/internal/store"
)
```

Then append to `internal/display/display_test.go`:

```go
func TestPrintRecords_FullTable(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	pr := func(v float64) *store.PersonalRecord {
		return &store.PersonalRecord{RawValue: v, Date: date}
	}
	recs := store.Records{
		LongestDistanceM:   pr(50000),
		LongestDurationS:   pr(7200),
		MostElevationGainM: pr(1500),
		HighestAvgPowerW:   pr(250),
		HighestAvgSpeedMPS: pr(10.0),
		HighestAvgHRBPM:    pr(155),
		HighestMaxSpeedMPS: pr(20.0),
		MostCaloriesKcal:   pr(800),
		HighestAltitudeM:   pr(1200),
	}
	PrintRecords(&buf, recs, "all time", false, "metric")
	output := buf.String()

	for _, want := range []string{
		"Longest distance", "50.0 km", "2024-06-15",
		"Longest duration", "2h 00m 00s",
		"Most elevation gain", "1500 m",
		"Highest avg power", "250 W",
		"Highest avg speed", "36.0 km/h",
		"Highest avg HR", "155 bpm",
		"Highest max speed", "72.0 km/h",
		"Most calories", "800",
		"Highest altitude", "1200 m",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestPrintRecords_PartialTable(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	pr := func(v float64) *store.PersonalRecord {
		return &store.PersonalRecord{RawValue: v, Date: date}
	}
	recs := store.Records{
		LongestDistanceM:   pr(50000),
		LongestDurationS:   pr(7200),
		MostElevationGainM: pr(1500),
		HighestAvgPowerW:   nil, // no power data
		HighestAvgSpeedMPS: pr(10.0),
		HighestAvgHRBPM:    nil, // no HR data
		HighestMaxSpeedMPS: pr(20.0),
		MostCaloriesKcal:   nil, // no calorie data
		HighestAltitudeM:   nil, // no stream altitude data
	}
	PrintRecords(&buf, recs, "all time", false, "metric")
	output := buf.String()

	if !strings.Contains(output, "Longest distance") {
		t.Errorf("expected 'Longest distance' in output, got:\n%s", output)
	}
	for _, absent := range []string{"Highest avg power", "Highest avg HR", "Most calories", "Highest altitude"} {
		if strings.Contains(output, absent) {
			t.Errorf("expected %q to be absent from output:\n%s", absent, output)
		}
	}
}

func TestPrintRecords_EmptyDB(t *testing.T) {
	var buf bytes.Buffer
	PrintRecords(&buf, store.Records{}, "all time", false, "metric")
	output := buf.String()
	if !strings.Contains(output, "No rides imported yet") {
		t.Errorf("expected empty-DB message, got:\n%s", output)
	}
}

func TestPrintRecords_EmptyFilter(t *testing.T) {
	var buf bytes.Buffer
	PrintRecords(&buf, store.Records{}, "2099", false, "metric")
	output := buf.String()
	if !strings.Contains(output, "No rides found for the selected period") {
		t.Errorf("expected period message, got:\n%s", output)
	}
}

func TestPrintRecords_JSON(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	recs := store.Records{
		LongestDistanceM: &store.PersonalRecord{RawValue: 50000, Date: date},
	}
	PrintRecords(&buf, recs, "all time", true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"longest_distance_m"`) {
		t.Errorf("expected JSON key 'longest_distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "50000") {
		t.Errorf("expected value 50000 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, "null") {
		t.Errorf("expected null fields in JSON for absent records, got:\n%s", output)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/display/... -run "TestPrintRecords" -v
```

Expected: FAIL — `PrintRecords` is undefined.

- [ ] **Step 3: Implement `PrintRecords` in `internal/display/table.go`**

Ensure the existing imports in `table.go` include `"strconv"` (already present) and `"fmt"` (already present). No new imports needed — `store` is already imported for `PrintStats`.

Append to `internal/display/table.go`:

```go
// PrintRecords renders personal records to w. Nil fields in recs are omitted entirely.
// If all fields are nil, a message is printed instead of an empty table.
// If jsonOut is true, recs is serialised as JSON (nil fields appear as null).
func PrintRecords(w io.Writer, recs store.Records, label string, jsonOut bool, units string) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(recs)
		return
	}

	fmt.Fprintf(w, "Personal Records: %s\n\n", label)

	type recordRow struct {
		name   string
		pr     *store.PersonalRecord
		format func(v float64) string
	}

	rows := []recordRow{
		{
			name:   "Longest distance",
			pr:     recs.LongestDistanceM,
			format: func(v float64) string { return FormatDistance(v, units) },
		},
		{
			name:   "Longest duration",
			pr:     recs.LongestDurationS,
			format: func(v float64) string { return formatDuration(int(v)) },
		},
		{
			name:   "Most elevation gain",
			pr:     recs.MostElevationGainM,
			format: func(v float64) string { return FormatElevation(v, units) },
		},
		{
			name:   "Highest avg power",
			pr:     recs.HighestAvgPowerW,
			format: func(v float64) string { return fmt.Sprintf("%d W", int(v)) },
		},
		{
			name:   "Highest avg speed",
			pr:     recs.HighestAvgSpeedMPS,
			format: func(v float64) string { return formatSpeed(v, units) },
		},
		{
			name:   "Highest avg HR",
			pr:     recs.HighestAvgHRBPM,
			format: func(v float64) string { return fmt.Sprintf("%d bpm", int(v)) },
		},
		{
			name:   "Highest max speed",
			pr:     recs.HighestMaxSpeedMPS,
			format: func(v float64) string { return formatSpeed(v, units) },
		},
		{
			name:   "Most calories",
			pr:     recs.MostCaloriesKcal,
			format: func(v float64) string { return strconv.Itoa(int(v)) },
		},
		{
			name:   "Highest altitude",
			pr:     recs.HighestAltitudeM,
			format: func(v float64) string { return FormatElevation(v, units) },
		},
	}

	var tableRows [][]string
	for _, r := range rows {
		if r.pr == nil {
			continue
		}
		tableRows = append(tableRows, []string{
			r.name,
			r.format(r.pr.RawValue),
			r.pr.Date.Format("2006-01-02"),
		})
	}

	if len(tableRows) == 0 {
		if label == "all time" {
			fmt.Fprintln(w, "No rides imported yet — run 'paceline import <file>' to get started.")
		} else {
			fmt.Fprintln(w, "No rides found for the selected period.")
		}
		return
	}

	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header([]string{"Record", "Value", "Date"})
	for _, r := range tableRows {
		table.Append(r)
	}
	table.Render()
}
```

- [ ] **Step 4: Run display tests to verify they pass**

```bash
go test ./internal/display/... -run "TestPrintRecords" -v
```

Expected: all 5 `TestPrintRecords_*` tests PASS.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go
git commit -m "feat: add PrintRecords to display layer"
```

---

## Task 3: Command — `cmd/records.go`

**Files:**
- Create: `cmd/records.go`

- [ ] **Step 1: Create `cmd/records.go`**

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
)

func init() {
	rootCmd.AddCommand(recordsCmd)
	recordsCmd.Flags().IntVar(&recordsYear, "year", 0, "filter by year (e.g. 2024)")
	recordsCmd.Flags().IntVar(&recordsMonth, "month", 0, "filter by month (1-12)")
	recordsCmd.Flags().IntVar(&recordsWeek, "week", 0, "filter by ISO week number (1-53)")
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

	noFlags := recordsYear == 0 && recordsMonth == 0 && recordsWeek == 0

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

	recs, err := s.GetRecords(f)
	if err != nil {
		return err
	}

	display.PrintRecords(os.Stdout, recs, label, jsonOutput, cfg.Units)
	return nil
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build -o paceline .
```

Expected: exits 0, `paceline` binary produced.

- [ ] **Step 3: Verify the command appears in help**

```bash
./paceline --help
```

Expected: output includes `records` in the command list.

- [ ] **Step 4: Smoke test — empty database**

```bash
./paceline records
```

Expected output (empty DB):
```
Personal Records: all time

No rides imported yet — run 'paceline import <file>' to get started.
```

- [ ] **Step 5: Smoke test — with real data (import a sample file first if needed)**

```bash
./paceline import testdata/sample.fit
./paceline records
```

Expected: a table with at least the non-nullable records (Longest distance, Longest duration, Most elevation gain, Highest avg speed, Highest max speed) populated. Nullable rows (avg power, HR, calories, altitude) appear only if the sample file contains those fields.

- [ ] **Step 6: Smoke test — JSON output**

```bash
./paceline records --json
```

Expected: valid JSON with `"longest_distance_m"`, `"longest_duration_s"` etc. Absent records appear as `null`.

- [ ] **Step 7: Run full test suite**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/records.go
git commit -m "feat: add paceline records command"
```

---

## Done

All three tasks complete. The `paceline records` command is fully implemented with:
- 9 personal best categories, nullable ones omitted when absent
- Year / month / week filtering
- JSON output
- 10 tests (5 store, 5 display)
- No schema changes, no new dependencies
