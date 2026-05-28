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
	StartTime   string       `xml:"StartTime,attr"`
	TotalTime   float64      `xml:"TotalTimeSeconds"`
	Distance    float64      `xml:"DistanceMeters"`
	Calories    int          `xml:"Calories"`
	AvgHR       tcxHRValue   `xml:"AverageHeartRateBpm"`
	MaxHR       tcxHRValue   `xml:"MaximumHeartRateBpm"`
	Trackpoints []tcxPoint   `xml:"Track>Trackpoint"`
}

type tcxHRValue struct {
	Value int `xml:"Value"`
}

type tcxTPX struct {
	Watts int `xml:"http://www.garmin.com/xmlschemas/ActivityExtension/v2 Watts"`
}

type tcxExtensions struct {
	TPX tcxTPX `xml:"http://www.garmin.com/xmlschemas/ActivityExtension/v2 TPX"`
}

type tcxPoint struct {
	Time       string        `xml:"Time"`
	Lat        float64       `xml:"Position>LatitudeDegrees"`
	Lon        float64       `xml:"Position>LongitudeDegrees"`
	Altitude   float64       `xml:"AltitudeMeters"`
	Distance   float64       `xml:"DistanceMeters"`
	HR         tcxHRValue    `xml:"HeartRateBpm"`
	Cadence    int           `xml:"Cadence"`
	Extensions tcxExtensions `xml:"Extensions"`
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

	streams := make([]Stream, 0, len(allPoints))
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
		if pt.Extensions.TPX.Watts > 0 {
			pw := pt.Extensions.TPX.Watts
			s.PowerW = &pw
		}
		streams = append(streams, s)
	}

	return ride, streams, nil
}
