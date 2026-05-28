package parser

import (
	"bytes"
	"fmt"
	"math"

	"github.com/tormoder/fit"
)

// ParseFIT parses a .fit binary file into a Ride and stream points.
func ParseFIT(filename string, data []byte) (*Ride, []Stream, error) {
	f, err := fit.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("decode fit: %w", err)
	}

	activity, err := f.Activity()
	if err != nil {
		return nil, nil, fmt.Errorf("get fit activity: %w", err)
	}

	if len(activity.Sessions) == 0 {
		return nil, nil, fmt.Errorf("fit file has no sessions")
	}

	sess := activity.Sessions[0]

	ride := &Ride{
		Filename:     filename,
		RecordedAt:   sess.StartTime,
		SourceFormat: "fit",
	}

	// TotalDistance: stored as uint32, scaled /100 → metres
	ride.DistanceM = sess.GetTotalDistanceScaled()

	// TotalElapsedTime: stored as uint32, scaled /1000 → seconds
	ride.DurationS = int(sess.GetTotalElapsedTimeScaled())

	// TotalAscent: stored as uint16 in metres (no scale), sentinel 0xFFFF
	if sess.TotalAscent != 0xFFFF {
		ride.ElevationGainM = float64(sess.TotalAscent)
	}

	// AvgSpeed / MaxSpeed: stored as uint16, scaled /1000 → m/s
	ride.AvgSpeedMPS = sess.GetAvgSpeedScaled()
	ride.MaxSpeedMPS = sess.GetMaxSpeedScaled()

	// AvgHeartRate / MaxHeartRate: uint8, sentinel 0xFF
	if sess.AvgHeartRate != 0xFF {
		v := int(sess.AvgHeartRate)
		ride.AvgHRBPM = &v
	}
	if sess.MaxHeartRate != 0xFF {
		v := int(sess.MaxHeartRate)
		ride.MaxHRBPM = &v
	}

	// AvgPower / MaxPower: uint16, sentinel 0xFFFF
	if sess.AvgPower != 0xFFFF && sess.AvgPower > 0 {
		v := int(sess.AvgPower)
		ride.AvgPowerW = &v
	}
	if sess.MaxPower != 0xFFFF && sess.MaxPower > 0 {
		v := int(sess.MaxPower)
		ride.MaxPowerW = &v
	}

	// TotalCalories: uint16, sentinel 0xFFFF
	if sess.TotalCalories != 0xFFFF && sess.TotalCalories > 0 {
		v := int(sess.TotalCalories)
		ride.Calories = &v
	}

	// Build stream points from Records
	start := sess.StartTime
	streams := make([]Stream, 0, len(activity.Records))
	for _, rec := range activity.Records {
		elapsed := int(rec.Timestamp.Sub(start).Seconds())

		s := Stream{
			Timestamp: rec.Timestamp,
			ElapsedS:  elapsed,
		}

		// Speed: uint16, scaled /1000 → m/s
		if rec.Speed != 0xFFFF {
			v := rec.GetSpeedScaled()
			if v > 0 {
				s.SpeedMPS = &v
			}
		} else if rec.EnhancedSpeed != 0xFFFFFFFF {
			v := rec.GetEnhancedSpeedScaled()
			if v > 0 {
				s.SpeedMPS = &v
			}
		}

		// HeartRate: uint8, sentinel 0xFF
		if rec.HeartRate != 0xFF && rec.HeartRate > 0 {
			v := int(rec.HeartRate)
			s.HRBPM = &v
		}

		// Power: uint16, sentinel 0xFFFF
		if rec.Power != 0xFFFF && rec.Power > 0 {
			v := int(rec.Power)
			s.PowerW = &v
		}

		// Cadence: uint8, sentinel 0xFF
		if rec.Cadence != 0xFF && rec.Cadence > 0 {
			v := int(rec.Cadence)
			s.CadenceRPM = &v
		}

		// Altitude: uint16, scaled /5 - 500 → metres; sentinel 0xFFFF
		// Prefer EnhancedAltitude if available
		if rec.EnhancedAltitude != 0xFFFFFFFF {
			v := rec.GetEnhancedAltitudeScaled()
			if !math.IsNaN(v) && v > 0 {
				s.AltitudeM = &v
			}
		} else if rec.Altitude != 0xFFFF {
			v := rec.GetAltitudeScaled()
			if !math.IsNaN(v) && v > 0 {
				s.AltitudeM = &v
			}
		}

		// Position: Latitude/Longitude stored as semicircles; Invalid() returns true if unset
		if !rec.PositionLat.Invalid() && !rec.PositionLong.Invalid() {
			lat := rec.PositionLat.Degrees()
			lon := rec.PositionLong.Degrees()
			if !math.IsNaN(lat) && !math.IsNaN(lon) {
				s.Lat = &lat
				s.Lon = &lon
			}
		}

		streams = append(streams, s)
	}

	return ride, streams, nil
}
