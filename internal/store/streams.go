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

	available := make([]string, 0)
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
	points := make([]parser.Stream, 0)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}
