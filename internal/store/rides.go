package store

import (
	"database/sql"
	"errors"
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
	var id int64
	err := s.db.QueryRow(`
		INSERT INTO rides (filename, recorded_at, distance_m, duration_s,
			elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_hr_bpm, max_hr_bpm, avg_power_w, max_power_w,
			calories, source_format)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (filename) DO NOTHING
		RETURNING id`,
		r.Filename, r.RecordedAt, r.DistanceM, r.DurationS,
		r.ElevationGainM, r.AvgSpeedMPS, r.MaxSpeedMPS,
		r.AvgHRBPM, r.MaxHRBPM, r.AvgPowerW, r.MaxPowerW,
		r.Calories, r.SourceFormat,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil // filename already exists, skipped
	}
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
	r, err := scanRide(row)
	if errors.Is(err, sql.ErrNoRows) {
		return parser.Ride{}, fmt.Errorf("ride %d not found", id)
	}
	return r, err
}

// ListRides returns filtered rides (most recent first) and the total count.
// Each ride's Position reflects its global rank (1 = most recent) across all rides,
// even when filters are applied.
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

// DeleteRide deletes a ride and its associated stream data by ID.
// Returns an error if no ride with the given ID exists.
// Streams are deleted first to satisfy the FK constraint before removing the ride.
func (s *Store) DeleteRide(id int64) error {
	// Check ride exists before deleting.
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM rides WHERE id = ?`, id).Scan(&count); err != nil {
		return fmt.Errorf("check ride: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("ride %d not found", id)
	}

	// Delete streams first (FK: streams.ride_id → rides.id).
	if _, err := s.db.Exec(`DELETE FROM streams WHERE ride_id = ?`, id); err != nil {
		return fmt.Errorf("delete streams: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM rides WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete ride: %w", err)
	}
	return nil
}

// DeleteAll removes all rides and stream data from the database.
// Streams are deleted first to satisfy FK constraints.
func (s *Store) DeleteAll() error {
	if _, err := s.db.Exec(`DELETE FROM streams`); err != nil {
		return fmt.Errorf("delete all streams: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM rides`); err != nil {
		return fmt.Errorf("delete all rides: %w", err)
	}
	return nil
}

type rideScanner interface {
	Scan(dest ...any) error
}

// applyNullableRideFields populates nullable sensor fields on r from the scanned sql.NullInt64 values.
func applyNullableRideFields(r *parser.Ride, avgHR, maxHR, avgPower, maxPower, calories sql.NullInt64) {
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
	applyNullableRideFields(&r, avgHR, maxHR, avgPower, maxPower, calories)
	return r, nil
}

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
	applyNullableRideFields(&r, avgHR, maxHR, avgPower, maxPower, calories)
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
