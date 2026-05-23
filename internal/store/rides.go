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
