package store

import (
	"fmt"
	"math"

	"github.com/HGClarke/paceline/internal/parser"
)

// PowerCurvePoint holds a duration and the mean maximal power for that duration.
type PowerCurvePoint struct {
	DurationS int
	PowerW    int
}

// PowerCurve holds the table (7 canonical durations) and chart (~50 log-spaced) MMP results.
type PowerCurve struct {
	Table []PowerCurvePoint
	Chart []PowerCurvePoint
}

var tableDurations = []int{5, 30, 60, 300, 600, 1200, 3600}

// GetPowerCurve computes the mean maximal power curve for a ride.
// Returns an error only for DB failures; if no power data exists the curves are empty.
func (s *Store) GetPowerCurve(rideID int64) (PowerCurve, error) {
	points, err := s.GetStreams(rideID, "power")
	if err != nil {
		return PowerCurve{}, err
	}
	if len(points) == 0 {
		return PowerCurve{}, nil
	}

	maxElapsed := points[len(points)-1].ElapsedS

	chartDurations := logSpacedDurations(5, maxElapsed, 50)
	allDurations := mergeDurations(tableDurations, chartDurations, maxElapsed)

	mmpMap := computeMMP(points, allDurations)

	var table []PowerCurvePoint
	for _, d := range tableDurations {
		if d > maxElapsed {
			break
		}
		if w, ok := mmpMap[d]; ok {
			table = append(table, PowerCurvePoint{DurationS: d, PowerW: w})
		}
	}

	var chart []PowerCurvePoint
	for _, d := range chartDurations {
		if w, ok := mmpMap[d]; ok {
			chart = append(chart, PowerCurvePoint{DurationS: d, PowerW: w})
		}
	}

	return PowerCurve{Table: table, Chart: chart}, nil
}

// logSpacedDurations returns ~n integer durations log-spaced between minS and maxS (inclusive).
func logSpacedDurations(minS, maxS, n int) []int {
	if maxS <= minS {
		return []int{minS}
	}
	logMin := math.Log(float64(minS))
	logMax := math.Log(float64(maxS))
	seen := make(map[int]bool)
	var out []int
	for i := 0; i < n; i++ {
		t := logMin + (logMax-logMin)*float64(i)/float64(n-1)
		d := int(math.Round(math.Exp(t)))
		if d < minS {
			d = minS
		}
		if d > maxS {
			d = maxS
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// mergeDurations combines two duration slices, deduplicating and keeping only those ≤ maxElapsed.
func mergeDurations(a, b []int, maxElapsed int) []int {
	seen := make(map[int]bool)
	var out []int
	for _, d := range a {
		if d <= maxElapsed && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	for _, d := range b {
		if d <= maxElapsed && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// computeMMP slides a window over the stream for each duration and returns the max avg power.
// Uses elapsed time (not index) so gaps in the recording are handled correctly.
func computeMMP(points []parser.Stream, durations []int) map[int]int {
	result := make(map[int]int, len(durations))

	n := len(points)
	prefix := make([]float64, n+1)
	for i, p := range points {
		v := 0.0
		if p.PowerW != nil {
			v = float64(*p.PowerW)
		}
		prefix[i+1] = prefix[i] + v
	}

	for _, d := range durations {
		best := -1.0
		left := 0
		for right := 0; right < n; right++ {
			for left < right && points[right].ElapsedS-points[left].ElapsedS > d {
				left++
			}
			span := points[right].ElapsedS - points[left].ElapsedS
			if span < d && right < n-1 {
				continue
			}
			count := right - left + 1
			if count <= 0 {
				continue
			}
			avg := (prefix[right+1] - prefix[left]) / float64(count)
			if avg > best {
				best = avg
			}
		}
		if best >= 0 {
			result[d] = int(math.Round(best))
		}
	}
	return result
}

// InsertStreams bulk-inserts stream data points for a ride.
func (s *Store) InsertStreams(streams []parser.Stream) error {
	if len(streams) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // deferred rollback; commit supersedes it on success

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

	// col is whitelisted by fieldColumn — not from user input directly.
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
	var pw, hr, spd, cad, alt int
	err := s.db.QueryRow(`
		SELECT
			COUNT(power_w),
			COUNT(hr_bpm),
			COUNT(speed_mps),
			COUNT(cadence_rpm),
			COUNT(altitude_m)
		FROM streams WHERE ride_id = ?`, rideID).Scan(&pw, &hr, &spd, &cad, &alt)
	if err != nil {
		return nil, fmt.Errorf("available fields: %w", err)
	}

	available := make([]string, 0)
	type entry struct {
		name string
		n    int
	}
	for _, e := range []entry{
		{"power", pw}, {"hr", hr}, {"speed", spd}, {"cadence", cad}, {"altitude", alt},
	} {
		if e.n > 0 {
			available = append(available, e.name)
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
		return nil, fmt.Errorf("scan streams: %w", err)
	}
	return points, nil
}
