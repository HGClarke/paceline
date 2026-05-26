package parser

import "time"

type Ride struct {
	ID             int64     `json:"-"`
	Position       int64     `json:"id"`
	Filename       string    `json:"filename"`
	RecordedAt     time.Time `json:"recorded_at"`
	DistanceM      float64   `json:"distance_m"`
	DurationS      int       `json:"duration_s"`
	ElevationGainM float64   `json:"elevation_gain_m"`
	AvgSpeedMPS    float64   `json:"avg_speed_mps"`
	MaxSpeedMPS    float64   `json:"max_speed_mps"`
	AvgHRBPM       *int      `json:"avg_hr_bpm,omitempty"`
	MaxHRBPM       *int      `json:"max_hr_bpm,omitempty"`
	AvgPowerW      *int      `json:"avg_power_w,omitempty"`
	MaxPowerW      *int      `json:"max_power_w,omitempty"`
	Calories       *int      `json:"calories,omitempty"`
	SourceFormat   string    `json:"source_format"`
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
