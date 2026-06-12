package store

import (
	"math"

	"github.com/HGClarke/paceline/internal/parser"
)

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
	if maxS <= minS || n <= 1 {
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
// Uses elapsed time (not index) so gaps are handled correctly, and averages are time-weighted
// so variable-rate streams (FIT smart-recording, GPX) produce correct results.
func computeMMP(points []parser.Stream, durations []int) map[int]int {
	result := make(map[int]int, len(durations))

	n := len(points)

	// Build time-weighted prefix sums. Each sample is weighted by the duration it
	// represents: the gap to the next sample, or 1 s for the final sample. This
	// ensures avg = watts-per-second rather than watts-per-sample.
	twPrefix := make([]float64, n+1) // cumulative (power × interval)
	wPrefix := make([]float64, n+1)  // cumulative interval seconds
	for i, p := range points {
		interval := 1.0
		if i < n-1 {
			if g := float64(points[i+1].ElapsedS - points[i].ElapsedS); g > 0 {
				interval = g
			}
		}
		v := 0.0
		if p.PowerW != nil {
			v = float64(*p.PowerW)
		}
		twPrefix[i+1] = twPrefix[i] + v*interval
		wPrefix[i+1] = wPrefix[i] + interval
	}

	for _, d := range durations {
		best := -1.0
		left := 0
		for right := 0; right < n; right++ {
			for left < right && points[right].ElapsedS-points[left].ElapsedS > d {
				left++
			}
			if points[right].ElapsedS-points[left].ElapsedS < d {
				continue
			}
			avg := (twPrefix[right+1] - twPrefix[left]) / (wPrefix[right+1] - wPrefix[left])
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
