# PacelineCLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that imports .fit/.gpx/.tcx ride files into a local DuckDB database and surfaces summary stats, time-series charts, and aggregated totals via terminal commands.

**Architecture:** Cobra command tree wires together three internal packages: `parser` (one file per format → shared Ride/Stream structs), `store` (DuckDB read/write), and `display` (tablewriter tables, asciigraph charts, bubbletea TUI). Commands never call parsers or display directly — they go through the store and display layer.

**Tech Stack:** Go 1.22+, cobra, go-duckdb (CGO required), tormoder/fit, tkrajina/gpxgo, olekukonko/tablewriter, guptarohit/asciigraph, charmbracelet/bubbletea, golang.org/x/term

---

## File Structure

```
paceline/
├── main.go
├── cmd/
│   ├── root.go       — cobra root, global --json flag
│   ├── import.go     — paceline import <file|dir>
│   ├── rides.go      — paceline rides [filters]
│   ├── ride.go       — paceline ride <id>
│   ├── stream.go     — paceline ride <id> stream [--field]
│   └── stats.go      — paceline stats [filters]
├── internal/
│   ├── parser/
│   │   ├── model.go      — Ride and Stream structs
│   │   ├── gpx.go        — .gpx → Ride + []Stream
│   │   ├── gpx_test.go
│   │   ├── tcx.go        — .tcx → Ride + []Stream
│   │   ├── tcx_test.go
│   │   ├── fit.go        — .fit → Ride + []Stream
│   │   └── fit_test.go
│   ├── store/
│   │   ├── db.go         — DuckDB open, schema migration
│   │   ├── db_test.go
│   │   ├── rides.go      — ride insert, list, get
│   │   ├── rides_test.go
│   │   ├── streams.go    — stream insert, query by ride+field
│   │   └── streams_test.go
│   └── display/
│       ├── table.go      — tablewriter: ride list, ride detail, stats
│       ├── chart.go      — asciigraph: stream line chart
│       └── tui.go        — bubbletea: interactive rides list
└── testdata/
    ├── sample.gpx
    ├── sample.tcx
    └── sample.fit        — copy a real .fit file here (see Task 9)
```

---

## Task 1: Project Bootstrap

**Files:**
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `go.mod` (via go mod init)

**Prerequisite:** CGO requires a C compiler. On macOS, run `xcode-select --install` if not already installed.

- [ ] **Step 1: Initialize the Go module**

```bash
cd /Users/hollandclarke/Desktop/PacelineCLI
go mod init github.com/hollandclarke/paceline
```

Expected: `go.mod` created with `module github.com/hollandclarke/paceline`

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/marcboeker/go-duckdb@latest
go get github.com/tormoder/fit@latest
go get github.com/tkrajina/gpxgo@latest
go get github.com/olekukonko/tablewriter@latest
go get github.com/guptarohit/asciigraph@latest
go get github.com/charmbracelet/bubbletea@latest
go get golang.org/x/term@latest
```

Expected: all packages added to `go.mod` and `go.sum`

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p cmd internal/parser internal/store internal/display testdata
```

- [ ] **Step 4: Write `cmd/root.go`**

```go
package cmd

import "github.com/spf13/cobra"

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "paceline",
	Short: "CLI for analyzing cycling ride data",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}
```

- [ ] **Step 5: Write `main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/hollandclarke/paceline/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Verify it compiles and runs**

```bash
go build ./... && ./paceline --help
```

Expected output includes `CLI for analyzing cycling ride data` and lists no subcommands yet.

- [ ] **Step 7: Commit**

```bash
git init
git add go.mod go.sum main.go cmd/root.go
git commit -m "feat: bootstrap project with cobra root command"
```

---

## Task 2: Data Model

**Files:**
- Create: `internal/parser/model.go`

- [ ] **Step 1: Write `internal/parser/model.go`**

```go
package parser

import "time"

type Ride struct {
	ID             int64
	Filename       string
	RecordedAt     time.Time
	DistanceM      float64
	DurationS      int
	ElevationGainM float64
	AvgSpeedMPS    float64
	MaxSpeedMPS    float64
	AvgHRBPM       *int
	MaxHRBPM       *int
	AvgPowerW      *int
	MaxPowerW      *int
	Calories       *int
	SourceFormat   string
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

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/parser/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/parser/model.go
git commit -m "feat: add shared Ride and Stream data model"
```

---

## Task 3: Store — DB Init and Schema

**Files:**
- Create: `internal/store/db.go`
- Create: `internal/store/db_test.go`

- [ ] **Step 1: Write the failing test in `internal/store/db_test.go`**

```go
package store_test

import (
	"testing"

	"github.com/hollandclarke/paceline/internal/store"
)

func TestOpen_CreatesSchema(t *testing.T) {
	s, err := store.Open("")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if err := s.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/store/... -run TestOpen_CreatesSchema -v
```

Expected: FAIL — `store` package does not exist yet.

- [ ] **Step 3: Write `internal/store/db.go`**

```go
package store

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

type Store struct {
	db *sql.DB
}

// Open opens (or creates) a DuckDB database at path.
// Pass "" or ":memory:" for an in-memory database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping() error { return s.db.Ping() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS rides_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS rides (
			id            BIGINT  DEFAULT nextval('rides_id_seq') PRIMARY KEY,
			filename      TEXT    UNIQUE NOT NULL,
			recorded_at   TIMESTAMP NOT NULL,
			distance_m    DOUBLE,
			duration_s    INTEGER,
			elevation_gain_m DOUBLE,
			avg_speed_mps DOUBLE,
			max_speed_mps DOUBLE,
			avg_hr_bpm    INTEGER,
			max_hr_bpm    INTEGER,
			avg_power_w   INTEGER,
			max_power_w   INTEGER,
			calories      INTEGER,
			source_format TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS streams (
			ride_id     BIGINT    NOT NULL REFERENCES rides(id),
			timestamp   TIMESTAMP NOT NULL,
			elapsed_s   INTEGER   NOT NULL,
			speed_mps   DOUBLE,
			hr_bpm      INTEGER,
			power_w     INTEGER,
			cadence_rpm INTEGER,
			altitude_m  DOUBLE,
			lat         DOUBLE,
			lon         DOUBLE
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Also add a `DefaultPath` helper at the bottom of `db.go`**

```go
import (
	"os"
	"path/filepath"
)

// DefaultPath returns ~/.paceline/data.db, creating the directory if needed.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".paceline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}
```

Note: merge this into the same `db.go` file (one file, add the imports to the existing import block).

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/store/... -run TestOpen_CreatesSchema -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/db.go internal/store/db_test.go
git commit -m "feat: add DuckDB store with schema migration"
```

---

## Task 4: Store — Ride Queries

**Files:**
- Create: `internal/store/rides.go`
- Create: `internal/store/rides_test.go`

- [ ] **Step 1: Write the failing tests in `internal/store/rides_test.go`**

```go
package store_test

import (
	"testing"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open("")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInsertAndGetRide(t *testing.T) {
	s := openTestStore(t)

	ride := parser.Ride{
		Filename:       "test.gpx",
		RecordedAt:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		DistanceM:      30000,
		DurationS:      3600,
		ElevationGainM: 500,
		AvgSpeedMPS:    8.3,
		MaxSpeedMPS:    15.0,
		SourceFormat:   "gpx",
	}

	id, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("InsertRide: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	got, err := s.GetRide(id)
	if err != nil {
		t.Fatalf("GetRide: %v", err)
	}
	if got.Filename != ride.Filename {
		t.Errorf("filename: got %q want %q", got.Filename, ride.Filename)
	}
	if got.DistanceM != ride.DistanceM {
		t.Errorf("distance: got %v want %v", got.DistanceM, ride.DistanceM)
	}
}

func TestInsertRide_Idempotent(t *testing.T) {
	s := openTestStore(t)

	ride := parser.Ride{
		Filename:     "dup.gpx",
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	}

	_, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	skipped, err := s.InsertRide(ride)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if skipped != 0 {
		t.Errorf("expected 0 (skipped) on duplicate, got %d", skipped)
	}
}

func TestListRides_Filters(t *testing.T) {
	s := openTestStore(t)

	rides := []parser.Ride{
		{Filename: "jan.gpx", RecordedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "feb.gpx", RecordedAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
		{Filename: "jan2.gpx", RecordedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	year, month := 2024, 1
	result, _, err := s.ListRides(store.RideFilters{Year: &year, Month: &month, Limit: 10, Page: 1})
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 January rides, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/store/... -run "TestInsertAndGetRide|TestInsertRide_Idempotent|TestListRides_Filters" -v
```

Expected: FAIL — `InsertRide`, `GetRide`, `ListRides`, `RideFilters` not defined.

- [ ] **Step 3: Write `internal/store/rides.go`**

```go
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
)

type RideFilters struct {
	Year  *int
	Month *int
	Date  *time.Time
	Page  int // 1-indexed
	Limit int // default 10
}

// InsertRide inserts a ride and returns its new ID.
// Returns 0 (no error) if the filename already exists (idempotent).
func (s *Store) InsertRide(r parser.Ride) (int64, error) {
	var existing int64
	err := s.db.QueryRow(`SELECT id FROM rides WHERE filename = ?`, r.Filename).Scan(&existing)
	if err == nil {
		return 0, nil // already imported
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("check existing: %w", err)
	}

	var id int64
	err = s.db.QueryRow(`
		INSERT INTO rides (filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		r.Filename, r.RecordedAt, r.DistanceM, r.DurationS,
		r.ElevationGainM, r.AvgSpeedMPS, r.MaxSpeedMPS,
		r.AvgHRBPM, r.MaxHRBPM, r.AvgPowerW, r.MaxPowerW,
		r.Calories, r.SourceFormat,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert ride: %w", err)
	}
	return id, nil
}

// GetRide returns a single ride by ID.
func (s *Store) GetRide(id int64) (parser.Ride, error) {
	row := s.db.QueryRow(`
		SELECT id, filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format
		FROM rides WHERE id = ?`, id)
	return scanRide(row)
}

// ListRides returns filtered rides (most recent first) and the total count.
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
		SELECT id, filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format
		FROM rides`+where+` ORDER BY recorded_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rides: %w", err)
	}
	defer rows.Close()

	var rides []parser.Ride
	for rows.Next() {
		r, err := scanRide(rows)
		if err != nil {
			return nil, 0, err
		}
		rides = append(rides, r)
	}
	return rides, total, rows.Err()
}

func buildRideWhere(f RideFilters) (string, []any) {
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

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

type rideScanner interface {
	Scan(dest ...any) error
}

func scanRide(row rideScanner) (parser.Ride, error) {
	var r parser.Ride
	var avgHR, maxHR, avgPower, maxPower, calories sql.NullInt64
	err := row.Scan(
		&r.ID, &r.Filename, &r.RecordedAt,
		&r.DistanceM, &r.DurationS, &r.ElevationGainM,
		&r.AvgSpeedMPS, &r.MaxSpeedMPS,
		&avgHR, &maxHR, &avgPower, &maxPower, &calories,
		&r.SourceFormat,
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
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/store/... -run "TestInsertAndGetRide|TestInsertRide_Idempotent|TestListRides_Filters" -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat: add ride insert, get, and filtered list queries"
```

---

## Task 5: Store — Stream Queries

**Files:**
- Create: `internal/store/streams.go`
- Create: `internal/store/streams_test.go`

- [ ] **Step 1: Write the failing tests in `internal/store/streams_test.go`**

```go
package store_test

import (
	"testing"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
)

func insertTestRide(t *testing.T, s *store.Store, filename string) int64 {
	t.Helper()
	id, err := s.InsertRide(parser.Ride{
		Filename:     filename,
		RecordedAt:   time.Now(),
		SourceFormat: "gpx",
	})
	if err != nil {
		t.Fatalf("insertTestRide: %v", err)
	}
	return id
}

func TestInsertAndGetStreams(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "stream_test.gpx")

	power := 200
	hr := 150
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, PowerW: &power, HRBPM: &hr},
		{RideID: rideID, Timestamp: time.Now().Add(time.Second), ElapsedS: 1, PowerW: &power, HRBPM: &hr},
	}

	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	got, err := s.GetStreams(rideID, "power")
	if err != nil {
		t.Fatalf("GetStreams: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 streams, got %d", len(got))
	}
	if got[0].PowerW == nil || *got[0].PowerW != 200 {
		t.Errorf("expected power 200, got %v", got[0].PowerW)
	}
}

func TestGetStreams_FieldFallback(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "fallback_test.gpx")

	hr := 155
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	// power requested but not present — should return empty
	got, err := s.GetStreams(rideID, "power")
	if err != nil {
		t.Fatalf("GetStreams power: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 power streams, got %d", len(got))
	}

	// hr is present
	got, err = s.GetStreams(rideID, "hr")
	if err != nil {
		t.Fatalf("GetStreams hr: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 hr stream, got %d", len(got))
	}
}

func TestAvailableFields(t *testing.T) {
	s := openTestStore(t)
	rideID := insertTestRide(t, s, "fields_test.gpx")

	hr := 140
	streams := []parser.Stream{
		{RideID: rideID, Timestamp: time.Now(), ElapsedS: 0, HRBPM: &hr},
	}
	if err := s.InsertStreams(streams); err != nil {
		t.Fatalf("InsertStreams: %v", err)
	}

	fields, err := s.AvailableFields(rideID)
	if err != nil {
		t.Fatalf("AvailableFields: %v", err)
	}
	if len(fields) != 1 || fields[0] != "hr" {
		t.Errorf("expected [hr], got %v", fields)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/store/... -run "TestInsertAndGetStreams|TestGetStreams_FieldFallback|TestAvailableFields" -v
```

Expected: FAIL — `InsertStreams`, `GetStreams`, `AvailableFields` not defined.

- [ ] **Step 3: Write `internal/store/streams.go`**

```go
package store

import (
	"fmt"

	"github.com/hollandclarke/paceline/internal/parser"
)

// InsertStreams bulk-inserts stream data points for a ride.
func (s *Store) InsertStreams(streams []parser.Stream) error {
	if len(streams) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO streams (ride_id, timestamp, elapsed_s, speed_mps, hr_bpm,
			power_w, cadence_rpm, altitude_m, lat, lon)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare stream insert: %w", err)
	}
	defer stmt.Close()

	for _, p := range streams {
		if _, err := stmt.Exec(
			p.RideID, p.Timestamp, p.ElapsedS,
			p.SpeedMPS, p.HRBPM, p.PowerW,
			p.CadenceRPM, p.AltitudeM, p.Lat, p.Lon,
		); err != nil {
			return fmt.Errorf("insert stream point: %w", err)
		}
	}
	return tx.Commit()
}

// GetStreams returns stream points for a ride filtered to rows where the
// requested field is non-null. field is one of: power, hr, speed, cadence, altitude.
// Points are ordered by elapsed_s ascending.
func (s *Store) GetStreams(rideID int64, field string) ([]parser.Stream, error) {
	col, ok := fieldColumn(field)
	if !ok {
		return nil, fmt.Errorf("unknown field %q: valid fields are power, hr, speed, cadence, altitude", field)
	}

	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT ride_id, timestamp, elapsed_s, speed_mps, hr_bpm,
			power_w, cadence_rpm, altitude_m, lat, lon
		FROM streams
		WHERE ride_id = ? AND %s IS NOT NULL
		ORDER BY elapsed_s ASC`, col), rideID)
	if err != nil {
		return nil, fmt.Errorf("get streams: %w", err)
	}
	defer rows.Close()
	return scanStreams(rows)
}

// AvailableFields returns which stream fields have at least one non-null value for a ride.
func (s *Store) AvailableFields(rideID int64) ([]string, error) {
	type check struct {
		name string
		col  string
	}
	checks := []check{
		{"power", "power_w"},
		{"hr", "hr_bpm"},
		{"speed", "speed_mps"},
		{"cadence", "cadence_rpm"},
		{"altitude", "altitude_m"},
	}

	var available []string
	for _, c := range checks {
		var n int
		err := s.db.QueryRow(fmt.Sprintf(
			`SELECT COUNT(*) FROM streams WHERE ride_id = ? AND %s IS NOT NULL`, c.col,
		), rideID).Scan(&n)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			available = append(available, c.name)
		}
	}
	return available, nil
}

func fieldColumn(field string) (string, bool) {
	m := map[string]string{
		"power":    "power_w",
		"hr":       "hr_bpm",
		"speed":    "speed_mps",
		"cadence":  "cadence_rpm",
		"altitude": "altitude_m",
	}
	col, ok := m[field]
	return col, ok
}

func scanStreams(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]parser.Stream, error) {
	var points []parser.Stream
	for rows.Next() {
		var p parser.Stream
		var speed, alt, lat, lon *float64
		var hr, power, cadence *int
		if err := rows.Scan(
			&p.RideID, &p.Timestamp, &p.ElapsedS,
			&speed, &hr, &power, &cadence, &alt, &lat, &lon,
		); err != nil {
			return nil, fmt.Errorf("scan stream: %w", err)
		}
		p.SpeedMPS = speed
		p.HRBPM = hr
		p.PowerW = power
		p.CadenceRPM = cadence
		p.AltitudeM = alt
		p.Lat = lat
		p.Lon = lon
		points = append(points, p)
	}
	return points, rows.Err()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/store/... -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/streams.go internal/store/streams_test.go
git commit -m "feat: add stream insert and field queries"
```

---

## Task 6: Store — Stats Queries

**Files:**
- Modify: `internal/store/rides.go` (add `GetStats`)
- Modify: `internal/store/rides_test.go` (add stats test)

- [ ] **Step 1: Write the failing test (append to `internal/store/rides_test.go`)**

```go
func TestGetStats_DefaultsToCurrentMonth(t *testing.T) {
	s := openTestStore(t)

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	rides := []parser.Ride{
		{Filename: "this_month.gpx", RecordedAt: now, DistanceM: 30000, DurationS: 3600, ElevationGainM: 400, SourceFormat: "gpx"},
		{Filename: "last_year.gpx", RecordedAt: now.AddDate(-1, 0, 0), DistanceM: 20000, DurationS: 2400, ElevationGainM: 200, SourceFormat: "gpx"},
	}
	for _, r := range rides {
		if _, err := s.InsertRide(r); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	stats, err := s.GetStats(store.StatsFilters{Year: &year, Month: &month})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.RideCount != 1 {
		t.Errorf("expected 1 ride this month, got %d", stats.RideCount)
	}
	if stats.TotalDistanceM != 30000 {
		t.Errorf("expected distance 30000, got %v", stats.TotalDistanceM)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/store/... -run TestGetStats -v
```

Expected: FAIL — `GetStats`, `StatsFilters`, `Stats` not defined.

- [ ] **Step 3: Append to `internal/store/rides.go`**

```go
type StatsFilters struct {
	Year  *int
	Month *int
	Week  *int
}

type Stats struct {
	RideCount       int
	TotalDistanceM  float64
	TotalDurationS  int
	TotalElevationM float64
}

func (s *Store) GetStats(f StatsFilters) (Stats, error) {
	where, args := buildStatsWhere(f)
	row := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(distance_m), 0),
			COALESCE(SUM(duration_s), 0),
			COALESCE(SUM(elevation_gain_m), 0)
		FROM rides`+where, args...)

	var st Stats
	if err := row.Scan(&st.RideCount, &st.TotalDistanceM, &st.TotalDurationS, &st.TotalElevationM); err != nil {
		return st, fmt.Errorf("get stats: %w", err)
	}
	return st, nil
}

func buildStatsWhere(f StatsFilters) (string, []any) {
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
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./internal/store/... -run TestGetStats -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/rides.go internal/store/rides_test.go
git commit -m "feat: add stats aggregation query with year/month/week filters"
```

---

## Task 7: Display — Tables and Charts

**Files:**
- Create: `internal/display/table.go`
- Create: `internal/display/chart.go`

- [ ] **Step 1: Write `internal/display/table.go`**

```go
package display

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
)

// PrintRideList renders a table of rides to w. If jsonOut is true, emits JSON instead.
func PrintRideList(w io.Writer, rides []parser.Ride, total, page, limit int, jsonOut bool) {
	if jsonOut {
		json.NewEncoder(w).Encode(rides)
		return
	}
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
	table.SetBorder(false)
	for _, r := range rides {
		table.Append([]string{
			strconv.FormatInt(r.ID, 10),
			r.RecordedAt.Format("2006-01-02"),
			fmt.Sprintf("%.1f km", r.DistanceM/1000),
			formatDuration(r.DurationS),
			fmt.Sprintf("%.0f m", r.ElevationGainM),
			fmt.Sprintf("%.1f km/h", r.AvgSpeedMPS*3.6),
		})
	}
	table.Render()
	pages := int(math.Ceil(float64(total) / float64(limit)))
	if pages > 1 {
		fmt.Fprintf(w, "\nPage %d of %d  —  run with --page=%d for next\n", page, pages, page+1)
	}
}

// PrintRideDetail renders a single ride's full summary to w.
func PrintRideDetail(w io.Writer, r parser.Ride, jsonOut bool) {
	if jsonOut {
		json.NewEncoder(w).Encode(r)
		return
	}
	table := tablewriter.NewWriter(w)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	rows := [][]string{
		{"Date", r.RecordedAt.Format(time.RFC1123)},
		{"Distance", fmt.Sprintf("%.2f km", r.DistanceM/1000)},
		{"Duration", formatDuration(r.DurationS)},
		{"Elevation Gain", fmt.Sprintf("%.0f m", r.ElevationGainM)},
		{"Avg Speed", fmt.Sprintf("%.1f km/h", r.AvgSpeedMPS*3.6)},
		{"Max Speed", fmt.Sprintf("%.1f km/h", r.MaxSpeedMPS*3.6)},
		{"Format", r.SourceFormat},
	}
	if r.AvgHRBPM != nil {
		rows = append(rows, []string{"Avg HR", fmt.Sprintf("%d bpm", *r.AvgHRBPM)})
	}
	if r.MaxHRBPM != nil {
		rows = append(rows, []string{"Max HR", fmt.Sprintf("%d bpm", *r.MaxHRBPM)})
	}
	if r.AvgPowerW != nil {
		rows = append(rows, []string{"Avg Power", fmt.Sprintf("%d W", *r.AvgPowerW)})
	}
	if r.MaxPowerW != nil {
		rows = append(rows, []string{"Max Power", fmt.Sprintf("%d W", *r.MaxPowerW)})
	}
	if r.Calories != nil {
		rows = append(rows, []string{"Calories", strconv.Itoa(*r.Calories)})
	}
	table.AppendBulk(rows)
	table.Render()
}

// PrintStats renders aggregated stats to w.
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool) {
	if jsonOut {
		json.NewEncoder(w).Encode(st)
		return
	}
	fmt.Fprintf(w, "Stats: %s\n\n", label)
	table := tablewriter.NewWriter(w)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.AppendBulk([][]string{
		{"Rides", strconv.Itoa(st.RideCount)},
		{"Total Distance", fmt.Sprintf("%.1f km", st.TotalDistanceM/1000)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", fmt.Sprintf("%.0f m", st.TotalElevationM)},
	})
	table.Render()
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	return fmt.Sprintf("%dm %02ds", m, s)
}
```

- [ ] **Step 2: Write `internal/display/chart.go`**

```go
package display

import (
	"fmt"
	"io"

	"github.com/guptarohit/asciigraph"
	"github.com/hollandclarke/paceline/internal/parser"
)

// PrintStreamChart renders a terminal line chart for the given stream points.
// field is used only for the chart caption.
func PrintStreamChart(w io.Writer, points []parser.Stream, field string) {
	data := extractFieldValues(points, field)
	if len(data) == 0 {
		fmt.Fprintf(w, "No %s data to display.\n", field)
		return
	}

	caption := fmt.Sprintf("%s over time (%d points)", field, len(data))
	chart := asciigraph.Plot(data, asciigraph.Height(15), asciigraph.Width(80), asciigraph.Caption(caption))
	fmt.Fprintln(w, chart)
}

func extractFieldValues(points []parser.Stream, field string) []float64 {
	var vals []float64
	for _, p := range points {
		switch field {
		case "power":
			if p.PowerW != nil {
				vals = append(vals, float64(*p.PowerW))
			}
		case "hr":
			if p.HRBPM != nil {
				vals = append(vals, float64(*p.HRBPM))
			}
		case "speed":
			if p.SpeedMPS != nil {
				vals = append(vals, *p.SpeedMPS*3.6) // convert to km/h
			}
		case "cadence":
			if p.CadenceRPM != nil {
				vals = append(vals, float64(*p.CadenceRPM))
			}
		case "altitude":
			if p.AltitudeM != nil {
				vals = append(vals, *p.AltitudeM)
			}
		}
	}
	return vals
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/display/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/display/table.go internal/display/chart.go
git commit -m "feat: add tablewriter and asciigraph display helpers"
```

---

## Task 8: Display — Interactive TUI

**Files:**
- Create: `internal/display/tui.go`

- [ ] **Step 1: Write `internal/display/tui.go`**

```go
package display

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
	"github.com/hollandclarke/paceline/internal/parser"
)

// IsTTY reports whether stdout is an interactive terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

type ridesModel struct {
	rides    []parser.Ride
	cursor   int
	page     int
	limit    int
	total    int
	selected *parser.Ride
	loadPage func(page int) ([]parser.Ride, int, error)
}

func (m ridesModel) Init() tea.Cmd { return nil }

func (m ridesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rides)-1 {
				m.cursor++
			}
		case "n", "right":
			maxPage := (m.total + m.limit - 1) / m.limit
			if m.page < maxPage {
				m.page++
				rides, total, _ := m.loadPage(m.page)
				m.rides = rides
				m.total = total
				m.cursor = 0
			}
		case "p", "left":
			if m.page > 1 {
				m.page--
				rides, total, _ := m.loadPage(m.page)
				m.rides = rides
				m.total = total
				m.cursor = 0
			}
		case "enter":
			if len(m.rides) > 0 {
				r := m.rides[m.cursor]
				m.selected = &r
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m ridesModel) View() string {
	var sb strings.Builder
	maxPage := (m.total + m.limit - 1) / m.limit
	fmt.Fprintf(&sb, "Rides — Page %d of %d  (↑/↓ move, n/p page, enter select, q quit)\n\n", m.page, maxPage)

	header := fmt.Sprintf("  %-4s  %-12s  %-10s  %-10s  %-10s\n", "ID", "Date", "Distance", "Duration", "Elevation")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("─", 60) + "\n")

	for i, r := range m.rides {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		fmt.Fprintf(&sb, "%s%-4d  %-12s  %-10s  %-10s  %-10s\n",
			cursor,
			r.ID,
			r.RecordedAt.Format("2006-01-02"),
			fmt.Sprintf("%.1f km", r.DistanceM/1000),
			formatDuration(r.DurationS),
			fmt.Sprintf("%.0f m", r.ElevationGainM),
		)
	}
	return sb.String()
}

// RunRidesTUI launches the interactive rides list.
// loadPage is called when the user navigates to a different page.
// If the user selects a ride, the selected Ride is returned; otherwise nil.
func RunRidesTUI(w io.Writer, initialRides []parser.Ride, total, limit int, loadPage func(page int) ([]parser.Ride, int, error)) (*parser.Ride, error) {
	m := ridesModel{
		rides:    initialRides,
		total:    total,
		page:     1,
		limit:    limit,
		loadPage: loadPage,
	}
	p := tea.NewProgram(m, tea.WithOutput(w))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(ridesModel)
	return final.selected, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/display/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/display/tui.go
git commit -m "feat: add bubbletea interactive rides TUI with TTY detection"
```

---

## Task 9: Parser — GPX

**Files:**
- Create: `testdata/sample.gpx`
- Create: `internal/parser/gpx.go`
- Create: `internal/parser/gpx_test.go`

- [ ] **Step 1: Create `testdata/sample.gpx`**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test"
     xmlns="http://www.topografix.com/GPX/1/1"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <trk>
    <name>Test Ride</name>
    <trkseg>
      <trkpt lat="37.7749" lon="-122.4194">
        <ele>10.0</ele>
        <time>2024-01-15T10:00:00Z</time>
      </trkpt>
      <trkpt lat="37.7755" lon="-122.4180">
        <ele>15.0</ele>
        <time>2024-01-15T10:00:10Z</time>
      </trkpt>
      <trkpt lat="37.7760" lon="-122.4165">
        <ele>20.0</ele>
        <time>2024-01-15T10:00:20Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>
```

- [ ] **Step 2: Write the failing test in `internal/parser/gpx_test.go`**

```go
package parser_test

import (
	"os"
	"testing"

	"github.com/hollandclarke/paceline/internal/parser"
)

func TestParseGPX(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.gpx")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	ride, streams, err := parser.ParseGPX("sample.gpx", data)
	if err != nil {
		t.Fatalf("ParseGPX: %v", err)
	}

	if ride.SourceFormat != "gpx" {
		t.Errorf("source_format: got %q want %q", ride.SourceFormat, "gpx")
	}
	if ride.Filename != "sample.gpx" {
		t.Errorf("filename: got %q want %q", ride.Filename, "sample.gpx")
	}
	if len(streams) != 3 {
		t.Errorf("expected 3 stream points, got %d", len(streams))
	}
	if ride.RecordedAt.IsZero() {
		t.Error("RecordedAt is zero")
	}
	if ride.DistanceM <= 0 {
		t.Errorf("expected positive distance, got %v", ride.DistanceM)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./internal/parser/... -run TestParseGPX -v
```

Expected: FAIL — `ParseGPX` not defined.

- [ ] **Step 4: Write `internal/parser/gpx.go`**

```go
package parser

import (
	"fmt"
	"math"

	"github.com/tkrajina/gpxgo/gpx"
)

// ParseGPX parses a .gpx file into a Ride and stream points.
func ParseGPX(filename string, data []byte) (*Ride, []Stream, error) {
	g, err := gpx.ParseBytes(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse gpx: %w", err)
	}

	if len(g.Tracks) == 0 || len(g.Tracks[0].Segments) == 0 {
		return nil, nil, fmt.Errorf("gpx file has no track segments")
	}

	seg := g.Tracks[0].Segments[0]
	points := seg.Points
	if len(points) == 0 {
		return nil, nil, fmt.Errorf("gpx segment has no points")
	}

	ride := &Ride{
		Filename:     filename,
		RecordedAt:   points[0].Timestamp,
		SourceFormat: "gpx",
	}

	var streams []Stream
	var totalDist, totalElevGain, sumSpeed float64
	var maxSpeed float64
	start := points[0].Timestamp

	for i, pt := range points {
		elapsed := int(pt.Timestamp.Sub(start).Seconds())

		var speed *float64
		if i > 0 {
			prev := points[i-1]
			dt := pt.Timestamp.Sub(prev.Timestamp).Seconds()
			dd := haversine(prev.Latitude, prev.Longitude, pt.Latitude, pt.Longitude)
			totalDist += dd
			if dt > 0 {
				s := dd / dt
				speed = &s
				sumSpeed += s
				if s > maxSpeed {
					maxSpeed = s
				}
			}
			if pt.Elevation.Value() > prev.Elevation.Value() {
				totalElevGain += pt.Elevation.Value() - prev.Elevation.Value()
			}
		}

		alt := pt.Elevation.Value()
		lat := pt.Latitude
		lon := pt.Longitude

		streams = append(streams, Stream{
			Timestamp: pt.Timestamp,
			ElapsedS:  elapsed,
			SpeedMPS:  speed,
			AltitudeM: &alt,
			Lat:       &lat,
			Lon:       &lon,
		})
	}

	n := len(points)
	ride.DistanceM = totalDist
	ride.ElevationGainM = totalElevGain
	ride.MaxSpeedMPS = maxSpeed
	if n > 1 {
		ride.AvgSpeedMPS = sumSpeed / float64(n-1)
	}
	ride.DurationS = int(points[n-1].Timestamp.Sub(start).Seconds())

	return ride, streams, nil
}

// haversine returns the distance in metres between two lat/lon points.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in metres
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/parser/... -run TestParseGPX -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add testdata/sample.gpx internal/parser/gpx.go internal/parser/gpx_test.go
git commit -m "feat: add GPX parser with haversine distance calculation"
```

---

## Task 10: Parser — TCX

**Files:**
- Create: `testdata/sample.tcx`
- Create: `internal/parser/tcx.go`
- Create: `internal/parser/tcx_test.go`

- [ ] **Step 1: Create `testdata/sample.tcx`**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<TrainingCenterDatabase xmlns="http://www.garmin.com/xmlschemas/TrainingCenterDatabase/v2"
  xmlns:ax="http://www.garmin.com/xmlschemas/ActivityExtension/v2">
  <Activities>
    <Activity Sport="Biking">
      <Id>2024-01-15T10:00:00Z</Id>
      <Lap StartTime="2024-01-15T10:00:00Z">
        <TotalTimeSeconds>3600</TotalTimeSeconds>
        <DistanceMeters>30000</DistanceMeters>
        <Calories>500</Calories>
        <AverageHeartRateBpm><Value>150</Value></AverageHeartRateBpm>
        <MaximumHeartRateBpm><Value>180</Value></MaximumHeartRateBpm>
        <Track>
          <Trackpoint>
            <Time>2024-01-15T10:00:00Z</Time>
            <Position>
              <LatitudeDegrees>37.7749</LatitudeDegrees>
              <LongitudeDegrees>-122.4194</LongitudeDegrees>
            </Position>
            <AltitudeMeters>10.0</AltitudeMeters>
            <DistanceMeters>0</DistanceMeters>
            <HeartRateBpm><Value>148</Value></HeartRateBpm>
            <Cadence>80</Cadence>
            <Extensions>
              <ax:TPX><ax:Watts>210</ax:Watts></ax:TPX>
            </Extensions>
          </Trackpoint>
          <Trackpoint>
            <Time>2024-01-15T10:00:10Z</Time>
            <Position>
              <LatitudeDegrees>37.7755</LatitudeDegrees>
              <LongitudeDegrees>-122.4180</LongitudeDegrees>
            </Position>
            <AltitudeMeters>15.0</AltitudeMeters>
            <DistanceMeters>100</DistanceMeters>
            <HeartRateBpm><Value>152</Value></HeartRateBpm>
            <Cadence>82</Cadence>
            <Extensions>
              <ax:TPX><ax:Watts>220</ax:Watts></ax:TPX>
            </Extensions>
          </Trackpoint>
        </Track>
      </Lap>
    </Activity>
  </Activities>
</TrainingCenterDatabase>
```

- [ ] **Step 2: Write the failing test in `internal/parser/tcx_test.go`**

```go
package parser_test

import (
	"os"
	"testing"

	"github.com/hollandclarke/paceline/internal/parser"
)

func TestParseTCX(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.tcx")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	ride, streams, err := parser.ParseTCX("sample.tcx", data)
	if err != nil {
		t.Fatalf("ParseTCX: %v", err)
	}

	if ride.SourceFormat != "tcx" {
		t.Errorf("source_format: got %q want %q", ride.SourceFormat, "tcx")
	}
	if ride.DurationS != 3600 {
		t.Errorf("duration: got %d want 3600", ride.DurationS)
	}
	if ride.DistanceM != 30000 {
		t.Errorf("distance: got %v want 30000", ride.DistanceM)
	}
	if ride.Calories == nil || *ride.Calories != 500 {
		t.Errorf("calories: got %v want 500", ride.Calories)
	}
	if len(streams) != 2 {
		t.Errorf("expected 2 stream points, got %d", len(streams))
	}
	if streams[0].PowerW == nil || *streams[0].PowerW != 210 {
		t.Errorf("power: got %v want 210", streams[0].PowerW)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./internal/parser/... -run TestParseTCX -v
```

Expected: FAIL — `ParseTCX` not defined.

- [ ] **Step 4: Write `internal/parser/tcx.go`**

```go
package parser

import (
	"encoding/xml"
	"fmt"
	"time"
)

type tcxDB struct {
	Activities struct {
		Activity []tcxActivity `xml:"Activity"`
	} `xml:"Activities"`
}

type tcxActivity struct {
	Laps []tcxLap `xml:"Lap"`
}

type tcxLap struct {
	StartTime   string      `xml:"StartTime,attr"`
	TotalTime   float64     `xml:"TotalTimeSeconds"`
	Distance    float64     `xml:"DistanceMeters"`
	Calories    int         `xml:"Calories"`
	AvgHR       tcxHRValue  `xml:"AverageHeartRateBpm"`
	MaxHR       tcxHRValue  `xml:"MaximumHeartRateBpm"`
	Trackpoints []tcxPoint  `xml:"Track>Trackpoint"`
}

type tcxHRValue struct {
	Value int `xml:"Value"`
}

type tcxPoint struct {
	Time      string  `xml:"Time"`
	Lat       float64 `xml:"Position>LatitudeDegrees"`
	Lon       float64 `xml:"Position>LongitudeDegrees"`
	Altitude  float64 `xml:"AltitudeMeters"`
	Distance  float64 `xml:"DistanceMeters"`
	HR        tcxHRValue `xml:"HeartRateBpm"`
	Cadence   int     `xml:"Cadence"`
	PowerWatts int    `xml:"Extensions>TPX>Watts"`
}

// ParseTCX parses a .tcx file into a Ride and stream points.
func ParseTCX(filename string, data []byte) (*Ride, []Stream, error) {
	var db tcxDB
	if err := xml.Unmarshal(data, &db); err != nil {
		return nil, nil, fmt.Errorf("parse tcx: %w", err)
	}
	if len(db.Activities.Activity) == 0 || len(db.Activities.Activity[0].Laps) == 0 {
		return nil, nil, fmt.Errorf("tcx file has no activities or laps")
	}

	// Aggregate across all laps
	var totalDist, totalElevGain float64
	var totalTime float64
	var totalCals int
	var avgHRSum, maxHR int
	var lapCount int
	var allPoints []tcxPoint
	var startTime time.Time

	for _, lap := range db.Activities.Activity[0].Laps {
		totalDist += lap.Distance
		totalTime += lap.TotalTime
		totalCals += lap.Calories
		avgHRSum += lap.AvgHR.Value
		lapCount++
		if lap.MaxHR.Value > maxHR {
			maxHR = lap.MaxHR.Value
		}
		allPoints = append(allPoints, lap.Trackpoints...)
	}

	if len(allPoints) > 0 {
		t, err := time.Parse(time.RFC3339, allPoints[0].Time)
		if err != nil {
			return nil, nil, fmt.Errorf("parse start time: %w", err)
		}
		startTime = t
	}

	// Compute elevation gain from trackpoints
	for i := 1; i < len(allPoints); i++ {
		if allPoints[i].Altitude > allPoints[i-1].Altitude {
			totalElevGain += allPoints[i].Altitude - allPoints[i-1].Altitude
		}
	}

	ride := &Ride{
		Filename:       filename,
		RecordedAt:     startTime,
		DistanceM:      totalDist,
		DurationS:      int(totalTime),
		ElevationGainM: totalElevGain,
		SourceFormat:   "tcx",
	}
	if totalDist > 0 && totalTime > 0 {
		ride.AvgSpeedMPS = totalDist / totalTime
	}
	if totalCals > 0 {
		ride.Calories = &totalCals
	}
	if lapCount > 0 && avgHRSum > 0 {
		avg := avgHRSum / lapCount
		ride.AvgHRBPM = &avg
	}
	if maxHR > 0 {
		ride.MaxHRBPM = &maxHR
	}

	var streams []Stream
	for _, pt := range allPoints {
		t, err := time.Parse(time.RFC3339, pt.Time)
		if err != nil {
			continue
		}
		elapsed := int(t.Sub(startTime).Seconds())
		lat, lon, alt := pt.Lat, pt.Lon, pt.Altitude
		s := Stream{
			Timestamp: t,
			ElapsedS:  elapsed,
			AltitudeM: &alt,
		}
		if pt.Lat != 0 || pt.Lon != 0 {
			s.Lat = &lat
			s.Lon = &lon
		}
		if pt.HR.Value > 0 {
			hr := pt.HR.Value
			s.HRBPM = &hr
		}
		if pt.Cadence > 0 {
			cad := pt.Cadence
			s.CadenceRPM = &cad
		}
		if pt.PowerWatts > 0 {
			pw := pt.PowerWatts
			s.PowerW = &pw
		}
		streams = append(streams, s)
	}

	return ride, streams, nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/parser/... -run TestParseTCX -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add testdata/sample.tcx internal/parser/tcx.go internal/parser/tcx_test.go
git commit -m "feat: add TCX parser"
```

---

## Task 11: Parser — FIT

**Files:**
- Create: `internal/parser/fit.go`
- Create: `internal/parser/fit_test.go`
- Note: requires a real `.fit` file in `testdata/sample.fit`

- [ ] **Step 1: Obtain a sample .fit file**

Download a sample .fit file from the `tormoder/fit` library's test fixtures:
```bash
curl -L -o testdata/sample.fit \
  https://raw.githubusercontent.com/tormoder/fit/master/testdata/activity.fit
```

Or copy any activity .fit file exported from Garmin Connect / Wahoo into `testdata/sample.fit`.

- [ ] **Step 2: Write the failing test in `internal/parser/fit_test.go`**

```go
package parser_test

import (
	"os"
	"testing"

	"github.com/hollandclarke/paceline/internal/parser"
)

func TestParseFIT(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.fit")
	if err != nil {
		t.Skip("testdata/sample.fit not present — copy a .fit file to run this test")
	}

	ride, streams, err := parser.ParseFIT("sample.fit", data)
	if err != nil {
		t.Fatalf("ParseFIT: %v", err)
	}

	if ride.SourceFormat != "fit" {
		t.Errorf("source_format: got %q want fit", ride.SourceFormat)
	}
	if ride.RecordedAt.IsZero() {
		t.Error("RecordedAt is zero")
	}
	if len(streams) == 0 {
		t.Error("expected stream points, got none")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./internal/parser/... -run TestParseFIT -v
```

Expected: FAIL — `ParseFIT` not defined (or SKIP if no .fit file present).

- [ ] **Step 4: Write `internal/parser/fit.go`**

```go
package parser

import (
	"bytes"
	"fmt"
	"math"

	"github.com/tormoder/fit"
)

// ParseFIT parses a .fit activity file into a Ride and stream points.
func ParseFIT(filename string, data []byte) (*Ride, []Stream, error) {
	f, err := fit.Decode(bytes.NewReader(data), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("decode fit: %w", err)
	}

	activity, err := f.Activity()
	if err != nil {
		return nil, nil, fmt.Errorf("fit activity: %w", err)
	}

	if len(activity.Sessions) == 0 {
		return nil, nil, fmt.Errorf("fit file has no sessions")
	}

	sess := activity.Sessions[0]
	ride := &Ride{
		Filename:     filename,
		RecordedAt:   sess.StartTime,
		DistanceM:    float64(sess.TotalDistance) / 100, // semicircles → metres (already metres in fit lib)
		DurationS:    int(sess.TotalElapsedTime / 1000),
		SourceFormat: "fit",
	}

	// tormoder/fit stores distance in cm for some fields; check units
	// TotalDistance is in metres * 100 for some versions — verify with actual file
	if sess.TotalDistance > 0 {
		ride.DistanceM = float64(sess.TotalDistance) / 100
	}
	if sess.TotalAscent > 0 {
		ride.ElevationGainM = float64(sess.TotalAscent)
	}
	if sess.AvgSpeed > 0 {
		ride.AvgSpeedMPS = float64(sess.AvgSpeed) / 1000 // mm/s → m/s
	}
	if sess.MaxSpeed > 0 {
		ride.MaxSpeedMPS = float64(sess.MaxSpeed) / 1000
	}
	if sess.AvgHeartRate > 0 {
		hr := int(sess.AvgHeartRate)
		ride.AvgHRBPM = &hr
	}
	if sess.MaxHeartRate > 0 {
		hr := int(sess.MaxHeartRate)
		ride.MaxHRBPM = &hr
	}
	if sess.AvgPower > 0 && sess.AvgPower != math.MaxUint16 {
		pw := int(sess.AvgPower)
		ride.AvgPowerW = &pw
	}
	if sess.MaxPower > 0 && sess.MaxPower != math.MaxUint16 {
		pw := int(sess.MaxPower)
		ride.MaxPowerW = &pw
	}
	if sess.TotalCalories > 0 {
		cal := int(sess.TotalCalories)
		ride.Calories = &cal
	}

	var streams []Stream
	startTime := sess.StartTime
	for _, rec := range activity.Records {
		elapsed := int(rec.Timestamp.Sub(startTime).Seconds())
		s := Stream{
			Timestamp: rec.Timestamp,
			ElapsedS:  elapsed,
		}
		if rec.Speed > 0 {
			spd := float64(rec.Speed) / 1000 // mm/s → m/s
			s.SpeedMPS = &spd
		}
		if rec.HeartRate > 0 {
			hr := int(rec.HeartRate)
			s.HRBPM = &hr
		}
		if rec.Power > 0 && rec.Power != math.MaxUint16 {
			pw := int(rec.Power)
			s.PowerW = &pw
		}
		if rec.Cadence > 0 {
			cad := int(rec.Cadence)
			s.CadenceRPM = &cad
		}
		if rec.Altitude > 0 {
			alt := float64(rec.Altitude) / 5 // fit altitude unit → metres
			s.AltitudeM = &alt
		}
		if rec.PositionLat.Semicircles() != 0 {
			lat := rec.PositionLat.Degrees()
			lon := rec.PositionLong.Degrees()
			s.Lat = &lat
			s.Lon = &lon
		}
		streams = append(streams, s)
	}

	return ride, streams, nil
}
```

**Note:** The `tormoder/fit` library's field units can vary by .fit file version. If values look wrong during testing (e.g. speed 1000x off), check `sess.AvgSpeed` units by printing the raw value and adjusting the divisor. The constants above (`/1000` for speed, `/100` for distance) are the most common case.

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/parser/... -run TestParseFIT -v
```

Expected: PASS (or SKIP if no sample .fit file)

- [ ] **Step 6: Commit**

```bash
git add testdata/sample.fit internal/parser/fit.go internal/parser/fit_test.go
git commit -m "feat: add FIT binary file parser"
```

---

## Task 12: Command — import

**Files:**
- Create: `cmd/import.go`

- [ ] **Step 1: Write `cmd/import.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
)

var importCmd = &cobra.Command{
	Use:   "import <file|directory>",
	Short: "Import ride file(s) into the database",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
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

	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(path, e.Name()))
			}
		}
	} else {
		files = []string{path}
	}

	var imported, skipped int
	var errs []string

	for _, f := range files {
		if err := importFile(s, f); err != nil {
			skipped++
			errs = append(errs, fmt.Sprintf("%s — %v", filepath.Base(f), err))
		} else {
			imported++
		}
	}

	fmt.Printf("%d imported, %d skipped\n", imported, skipped)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  skipped: %s\n", e)
	}
	return nil
}

func importFile(s *store.Store, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	var ride *parser.Ride
	var streams []parser.Stream

	switch ext {
	case ".gpx":
		ride, streams, err = parser.ParseGPX(filename, data)
	case ".tcx":
		ride, streams, err = parser.ParseTCX(filename, data)
	case ".fit":
		ride, streams, err = parser.ParseFIT(filename, data)
	default:
		return fmt.Errorf("unsupported format %q", ext)
	}
	if err != nil {
		return err
	}

	id, err := s.InsertRide(*ride)
	if err != nil {
		return err
	}
	if id == 0 {
		fmt.Printf("  already imported: %s\n", filename)
		return nil
	}

	for i := range streams {
		streams[i].RideID = id
	}
	if err := s.InsertStreams(streams); err != nil {
		return fmt.Errorf("insert streams: %w", err)
	}

	fmt.Printf("  imported: %s (id=%d, %d points)\n", filename, id, len(streams))
	return nil
}
```

- [ ] **Step 2: Verify it compiles and the command appears**

```bash
go build ./... && ./paceline import --help
```

Expected: shows `Import ride file(s) into the database` usage.

- [ ] **Step 3: Smoke test with the sample GPX file**

```bash
./paceline import testdata/sample.gpx
```

Expected: `1 imported, 0 skipped` and `imported: sample.gpx (id=1, 3 points)`

- [ ] **Step 4: Smoke test idempotency**

```bash
./paceline import testdata/sample.gpx
```

Expected: `already imported: sample.gpx` and `1 imported, 0 skipped`

- [ ] **Step 5: Commit**

```bash
git add cmd/import.go
git commit -m "feat: add import command with idempotent file and directory import"
```

---

## Task 13: Command — rides

**Files:**
- Create: `cmd/rides.go`

- [ ] **Step 1: Write `cmd/rides.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
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
	ridesPage  int
	ridesLimit int
)

func init() {
	rootCmd.AddCommand(ridesCmd)
	ridesCmd.Flags().IntVar(&ridesYear, "year", 0, "filter by year (e.g. 2024)")
	ridesCmd.Flags().IntVar(&ridesMonth, "month", 0, "filter by month (1-12)")
	ridesCmd.Flags().StringVar(&ridesDate, "date", "", "filter by date (YYYY-MM-DD)")
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
		// Default year to current if month given without year
		if f.Year == nil {
			y := time.Now().Year()
			f.Year = &y
		}
		f.Month = &ridesMonth
	}
	if ridesDate != "" {
		t, err := time.Parse("2006-01-02", ridesDate)
		if err != nil {
			return fmt.Errorf("invalid date %q: use YYYY-MM-DD", ridesDate)
		}
		f.Date = &t
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
		selected, err := display.RunRidesTUI(os.Stdout, rides, total, ridesLimit, loadPage)
		if err != nil {
			return err
		}
		if selected != nil {
			display.PrintRideDetail(os.Stdout, *selected, false)
		}
		return nil
	}

	display.PrintRideList(os.Stdout, rides, total, ridesPage, ridesLimit, jsonOutput)
	return nil
}
```

Add `"github.com/hollandclarke/paceline/internal/parser"` to the import block in `cmd/rides.go`.

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 3: Smoke test**

```bash
./paceline rides
```

Expected: table showing imported rides (or TUI if running in terminal).

- [ ] **Step 4: Test JSON output**

```bash
./paceline rides --json
```

Expected: JSON array of ride objects.

- [ ] **Step 5: Commit**

```bash
git add cmd/rides.go
git commit -m "feat: add rides list command with TTY-aware TUI and filter flags"
```

---

## Task 14: Command — ride and stream

**Files:**
- Create: `cmd/ride.go`
- Create: `cmd/stream.go`

- [ ] **Step 1: Write `cmd/ride.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
)

// currentRideID is set by rideCmd's PersistentPreRunE so that the stream
// subcommand can read it without re-parsing the positional argument.
var currentRideID int64

var rideCmd = &cobra.Command{
	Use:   "ride <id>",
	Short: "Show summary stats for a specific ride",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ride ID %q: must be a number", args[0])
		}
		currentRideID = id
		return nil
	},
	RunE: runRide,
}

func init() {
	rootCmd.AddCommand(rideCmd)
	rideCmd.AddCommand(streamCmd)
}

func runRide(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	ride, err := s.GetRide(currentRideID)
	if err != nil {
		return fmt.Errorf("ride %d not found", currentRideID)
	}

	display.PrintRideDetail(os.Stdout, ride, jsonOutput)
	return nil
}
```

- [ ] **Step 2: Write `cmd/stream.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
)

var streamField string

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Show a time-series chart for a ride stream",
	RunE:  runStream,
}

func init() {
	streamCmd.Flags().StringVar(&streamField, "field", "", "field to chart: power, hr, speed, cadence, altitude")
}

func runStream(cmd *cobra.Command, args []string) error {
	// currentRideID is set by rideCmd.PersistentPreRunE before this runs.
	id := currentRideID

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	field := streamField
	if field == "" {
		available, err := s.AvailableFields(id)
		if err != nil {
			return err
		}
		for _, candidate := range []string{"power", "hr", "speed"} {
			for _, a := range available {
				if a == candidate {
					field = candidate
					break
				}
			}
			if field != "" {
				break
			}
		}
		if field == "" {
			return fmt.Errorf("ride %d has no stream data", id)
		}
	}

	points, err := s.GetStreams(id, field)
	if err != nil {
		return err
	}

	if len(points) == 0 {
		available, _ := s.AvailableFields(id)
		fmt.Fprintf(os.Stderr, "No %s data for ride %d. Available fields: %v\n", field, id, available)
		return nil
	}

	display.PrintStreamChart(os.Stdout, points, field)
	return nil
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Smoke test**

```bash
./paceline ride 1
./paceline ride 1 stream --field=altitude
```

Expected: ride detail table, then an ASCII altitude chart.

- [ ] **Step 5: Commit**

```bash
git add cmd/ride.go cmd/stream.go
git commit -m "feat: add ride detail and stream chart commands"
```

---

## Task 15: Command — stats

**Files:**
- Create: `cmd/stats.go`

- [ ] **Step 1: Write `cmd/stats.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
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
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsYear, "year", 0, "filter by year (e.g. 2024)")
	statsCmd.Flags().IntVar(&statsMonth, "month", 0, "filter by month (1-12)")
	statsCmd.Flags().IntVar(&statsWeek, "week", 0, "filter by ISO week number (1-53)")
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
	label := "current month"

	noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0
	if noFlags {
		// Default: current calendar month
		y, m := now.Year(), int(now.Month())
		f.Year = &y
		f.Month = &m
	} else {
		if statsYear != 0 {
			f.Year = &statsYear
			label = fmt.Sprintf("year %d", statsYear)
		}
		if statsMonth != 0 {
			f.Month = &statsMonth
			if f.Year == nil {
				y := now.Year()
				f.Year = &y
			}
			label = fmt.Sprintf("%s %02d", label, statsMonth)
		}
		if statsWeek != 0 {
			f.Week = &statsWeek
			if f.Year == nil {
				y := now.Year()
				f.Year = &y
			}
			label = fmt.Sprintf("%s week %d", label, statsWeek)
		}
	}

	st, err := s.GetStats(f)
	if err != nil {
		return err
	}

	display.PrintStats(os.Stdout, st, label, jsonOutput)
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 3: Smoke test**

```bash
./paceline stats
./paceline stats --year=2024 --month=1
./paceline stats --year=2024 --week=3
```

Expected: stats table with ride count, distance, duration, elevation for each filter.

- [ ] **Step 4: Run the full test suite**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 5: Final build verification**

```bash
go build -o paceline . && ./paceline --help
```

Expected: all subcommands listed: `import`, `rides`, `ride`, `stats`.

- [ ] **Step 6: Commit**

```bash
git add cmd/stats.go
git commit -m "feat: add stats command with year/month/week filters, defaults to current month"
```
