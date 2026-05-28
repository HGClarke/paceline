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

	streams := make([]Stream, 0, len(points))
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
