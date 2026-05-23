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
