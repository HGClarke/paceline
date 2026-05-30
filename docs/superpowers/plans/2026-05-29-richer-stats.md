# Richer Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `paceline stats` to show avg/max speed (always), avg/max power, and avg/max HR (when data exists), in addition to the existing ride count and totals.

**Architecture:** Two isolated changes in sequence — store layer first (extend `Stats` struct + SQL), then display layer (`PrintStats` renders new rows conditionally). No new files; no schema migration needed (all columns already in the `rides` table). TDD throughout: failing test → implementation → passing test → commit.

**Tech Stack:** Go, DuckDB (`database/sql` + `go-duckdb`), `tablewriter` for table rendering, `encoding/json` for JSON output.

---

## File Map

| File | Change |
|------|--------|
| `internal/store/rides.go` | Extend `Stats` struct (6 new fields); update `GetStats()` SQL to 10 columns + scan |
| `internal/store/rides_test.go` | Add `TestGetStats_NewFields_WithSensorData` and `TestGetStats_NewFields_NoSensorData` |
| `internal/display/table.go` | Update `PrintStats()` to render speed rows always, power/HR rows conditionally |
| `internal/display/display_test.go` | Add `TestPrintStats_WithPowerAndHR`, `TestPrintStats_NoPowerNoHR`; update `TestPrintStats_JSON` |

---

## Task 1: Write failing store tests for new Stats fields

**Files:**
- Modify: `internal/store/rides_test.go`

- [ ] **Step 1: Append two new tests to `internal/store/rides_test.go`**

Add after the last test in the file:

```go
func TestGetStats_NewFields_WithSensorData(t *testing.T) {
	s := openTestStore(t)

	avgHR, maxHR := 150, 175
	avgPower, maxPower := 220, 380
	rides := []parser.Ride{
		{
			Filename:     "ride1.gpx",
			RecordedAt:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			AvgSpeedMPS:  8.0,
			MaxSpeedMPS:  14.0,
			AvgHRBPM:     &avgHR,
			MaxHRBPM:     &maxHR,
			AvgPowerW:    &avgPower,
			MaxPowerW:    &maxPower,
			SourceFormat: "gpx",
		},
		{
			Filename:     "ride2.gpx",
			RecordedAt:   time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC),
			AvgSpeedMPS:  10.0,
			MaxSpeedMPS:  18.0,
			AvgHRBPM:     &avgHR,
			MaxHRBPM:     &maxHR,
			AvgPowerW:    &avgPower,
			MaxPowerW:    &maxPower,
			SourceFormat: "gpx",
		},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("InsertRide: %v", err)
		}
	}

	st, err := s.GetStats(store.StatsFilters{})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	// AVG(8.0, 10.0) = 9.0; MAX(14.0, 18.0) = 18.0
	if st.AvgSpeedMPS != 9.0 {
		t.Errorf("AvgSpeedMPS: got %v, want 9.0", st.AvgSpeedMPS)
	}
	if st.MaxSpeedMPS != 18.0 {
		t.Errorf("MaxSpeedMPS: got %v, want 18.0", st.MaxSpeedMPS)
	}
	if st.AvgPowerW == nil {
		t.Fatal("AvgPowerW: got nil, want non-nil")
	}
	if int(*st.AvgPowerW) != 220 {
		t.Errorf("AvgPowerW: got %v, want 220", *st.AvgPowerW)
	}
	if st.MaxPowerW == nil {
		t.Fatal("MaxPowerW: got nil, want non-nil")
	}
	if int(*st.MaxPowerW) != 380 {
		t.Errorf("MaxPowerW: got %v, want 380", *st.MaxPowerW)
	}
	if st.AvgHRBPM == nil {
		t.Fatal("AvgHRBPM: got nil, want non-nil")
	}
	if int(*st.AvgHRBPM) != 150 {
		t.Errorf("AvgHRBPM: got %v, want 150", *st.AvgHRBPM)
	}
	if st.MaxHRBPM == nil {
		t.Fatal("MaxHRBPM: got nil, want non-nil")
	}
	if int(*st.MaxHRBPM) != 175 {
		t.Errorf("MaxHRBPM: got %v, want 175", *st.MaxHRBPM)
	}
}

func TestGetStats_NewFields_NoSensorData(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.InsertRide(parser.Ride{
		Filename:     "no_sensors.gpx",
		RecordedAt:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		AvgSpeedMPS:  7.5,
		MaxSpeedMPS:  12.0,
		SourceFormat: "gpx",
	}); err != nil {
		t.Fatalf("InsertRide: %v", err)
	}

	st, err := s.GetStats(store.StatsFilters{})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if st.AvgSpeedMPS != 7.5 {
		t.Errorf("AvgSpeedMPS: got %v, want 7.5", st.AvgSpeedMPS)
	}
	if st.MaxSpeedMPS != 12.0 {
		t.Errorf("MaxSpeedMPS: got %v, want 12.0", st.MaxSpeedMPS)
	}
	if st.AvgPowerW != nil {
		t.Errorf("AvgPowerW: got %v, want nil", *st.AvgPowerW)
	}
	if st.MaxPowerW != nil {
		t.Errorf("MaxPowerW: got %v, want nil", *st.MaxPowerW)
	}
	if st.AvgHRBPM != nil {
		t.Errorf("AvgHRBPM: got %v, want nil", *st.AvgHRBPM)
	}
	if st.MaxHRBPM != nil {
		t.Errorf("MaxHRBPM: got %v, want nil", *st.MaxHRBPM)
	}
}
```

- [ ] **Step 2: Run the new tests — confirm they fail to compile**

```bash
go test ./internal/store/... -run "TestGetStats_NewFields" -v
```

Expected: compile error — `st.AvgSpeedMPS undefined` (field doesn't exist yet).

---

## Task 2: Extend `Stats` struct and `GetStats()` SQL

**Files:**
- Modify: `internal/store/rides.go`

- [ ] **Step 1: Replace the `Stats` struct**

In `internal/store/rides.go`, replace:

```go
type Stats struct {
	RideCount       int
	TotalDistanceM  float64
	TotalDurationS  int
	TotalElevationM float64
}
```

With:

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

- [ ] **Step 2: Replace the `GetStats()` function**

In `internal/store/rides.go`, replace the entire `GetStats` function:

```go
func (s *Store) GetStats(f StatsFilters) (Stats, error) {
	where, args := buildStatsWhere(f)
	row := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(distance_m), 0),
			COALESCE(SUM(duration_s), 0),
			COALESCE(SUM(elevation_gain_m), 0),
			COALESCE(AVG(avg_speed_mps), 0),
			COALESCE(MAX(max_speed_mps), 0),
			AVG(avg_power_w),
			CAST(MAX(max_power_w) AS DOUBLE),
			AVG(avg_hr_bpm),
			CAST(MAX(max_hr_bpm) AS DOUBLE)
		FROM rides`+where, args...)

	var st Stats
	var avgPower, maxPower, avgHR, maxHR sql.NullFloat64
	if err := row.Scan(
		&st.RideCount, &st.TotalDistanceM, &st.TotalDurationS, &st.TotalElevationM,
		&st.AvgSpeedMPS, &st.MaxSpeedMPS,
		&avgPower, &maxPower, &avgHR, &maxHR,
	); err != nil {
		return st, fmt.Errorf("get stats: %w", err)
	}
	if avgPower.Valid {
		st.AvgPowerW = &avgPower.Float64
	}
	if maxPower.Valid {
		st.MaxPowerW = &maxPower.Float64
	}
	if avgHR.Valid {
		st.AvgHRBPM = &avgHR.Float64
	}
	if maxHR.Valid {
		st.MaxHRBPM = &maxHR.Float64
	}
	return st, nil
}
```

- [ ] **Step 3: Run all store tests — confirm they all pass**

```bash
go test ./internal/store/... -v
```

Expected: all tests PASS, including the two new ones.

- [ ] **Step 4: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat(store): extend Stats with avg/max speed, power, and HR fields"
```

---

## Task 3: Write failing display tests for new PrintStats rows

**Files:**
- Modify: `internal/display/display_test.go`

- [ ] **Step 1: Update `TestPrintStats_JSON` to cover new JSON keys**

In `internal/display/display_test.go`, replace the existing `TestPrintStats_JSON` function:

```go
func TestPrintStats_JSON(t *testing.T) {
	var buf bytes.Buffer
	avgPower := 231.0
	st := store.Stats{
		RideCount:      5,
		TotalDistanceM: 100000,
		TotalDurationS: 18000,
		AvgSpeedMPS:    7.5,
		MaxSpeedMPS:    12.0,
		AvgPowerW:      &avgPower,
	}
	PrintStats(&buf, st, "all time", true, "metric")
	output := buf.String()
	for _, want := range []string{
		`"RideCount"`, `"AvgSpeedMPS"`, `"MaxSpeedMPS"`,
		`"AvgPowerW"`, `"MaxPowerW"`, `"AvgHRBPM"`, `"MaxHRBPM"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected JSON key %s in output, got:\n%s", want, output)
		}
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}
```

- [ ] **Step 2: Append two new display tests to `internal/display/display_test.go`**

Add after `TestPrintStats_JSON`:

```go
func TestPrintStats_WithPowerAndHR(t *testing.T) {
	var buf bytes.Buffer
	avgPower, maxPower := 231.0, 421.0
	avgHR, maxHR := 148.0, 183.0
	st := store.Stats{
		RideCount:       14,
		TotalDistanceM:  423700,
		TotalDurationS:  67320,
		TotalElevationM: 4210,
		AvgSpeedMPS:     7.89,
		MaxSpeedMPS:     14.47,
		AvgPowerW:       &avgPower,
		MaxPowerW:       &maxPower,
		AvgHRBPM:        &avgHR,
		MaxHRBPM:        &maxHR,
	}
	PrintStats(&buf, st, "all time", false, "metric")
	output := buf.String()
	for _, want := range []string{
		"Avg Speed", "Max Speed",
		"Avg Power", "231 W",
		"Max Power", "421 W",
		"Avg HR", "148 bpm",
		"Max HR", "183 bpm",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestPrintStats_NoPowerNoHR(t *testing.T) {
	var buf bytes.Buffer
	st := store.Stats{
		RideCount:      3,
		TotalDistanceM: 87200,
		TotalDurationS: 11640,
		AvgSpeedMPS:    7.5,
		MaxSpeedMPS:    12.0,
	}
	PrintStats(&buf, st, "all time", false, "metric")
	output := buf.String()
	for _, want := range []string{"Avg Speed", "Max Speed"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
	for _, absent := range []string{"Avg Power", "Max Power", "Avg HR", "Max HR"} {
		if strings.Contains(output, absent) {
			t.Errorf("expected %q to be absent from output:\n%s", absent, output)
		}
	}
}
```

- [ ] **Step 3: Run the new display tests — confirm they fail**

```bash
go test ./internal/display/... -run "TestPrintStats" -v
```

Expected: `TestPrintStats_WithPowerAndHR` and `TestPrintStats_NoPowerNoHR` FAIL ("Avg Speed" not found). `TestPrintStats_JSON` FAIL (new JSON keys not found).

---

## Task 4: Update `PrintStats()` to render new rows

**Files:**
- Modify: `internal/display/table.go`

- [ ] **Step 1: Replace the `PrintStats` function**

In `internal/display/table.go`, replace the entire `PrintStats` function:

```go
// PrintStats renders aggregated stats to w.
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool, units string) {
	if jsonOut {
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Fprintf(w, "Stats: %s\n\n", label)
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	rows := [][]string{
		{"Rides", strconv.Itoa(st.RideCount)},
		{"Total Distance", FormatDistance(st.TotalDistanceM, units)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", FormatElevation(st.TotalElevationM, units)},
		{"Avg Speed", formatSpeed(st.AvgSpeedMPS, units)},
		{"Max Speed", formatSpeed(st.MaxSpeedMPS, units)},
	}
	if st.AvgPowerW != nil {
		rows = append(rows, []string{"Avg Power", fmt.Sprintf("%d W", int(*st.AvgPowerW))})
	}
	if st.MaxPowerW != nil {
		rows = append(rows, []string{"Max Power", fmt.Sprintf("%d W", int(*st.MaxPowerW))})
	}
	if st.AvgHRBPM != nil {
		rows = append(rows, []string{"Avg HR", fmt.Sprintf("%d bpm", int(*st.AvgHRBPM))})
	}
	if st.MaxHRBPM != nil {
		rows = append(rows, []string{"Max HR", fmt.Sprintf("%d bpm", int(*st.MaxHRBPM))})
	}
	_ = table.Bulk(rows) // write errors are unrecoverable; discard return value
	_ = table.Render()   // write errors are unrecoverable; discard return value
}
```

- [ ] **Step 2: Run all display tests — confirm they all pass**

```bash
go test ./internal/display/... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Run the full test suite — confirm no regressions**

```bash
go test ./...
```

Expected: all packages PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go
git commit -m "feat(display): render avg/max speed, power, and HR in stats output"
```

---

## Task 5: Smoke test the binary

**Files:** none

- [ ] **Step 1: Build and run stats**

```bash
go build -o paceline . && ./paceline stats
```

Expected: table includes `Avg Speed` and `Max Speed` rows. Power/HR rows appear only if your local DB has rides with that sensor data.

- [ ] **Step 2: Verify JSON output**

```bash
./paceline stats --json
```

Expected: JSON object contains `"AvgSpeedMPS"`, `"MaxSpeedMPS"`, `"AvgPowerW"`, `"MaxPowerW"`, `"AvgHRBPM"`, `"MaxHRBPM"` keys.

- [ ] **Step 3: Clean up binary**

```bash
rm paceline
```
