package store

import "time"

type RideFilters struct {
	Year      *int
	Month     *int
	Date      *time.Time
	From      *time.Time
	To        *time.Time
	Page      int    // 1-indexed
	Limit     int    // default 10
	SortField string // "date", "distance", "duration", "elevation", "power", "speed"
	SortOrder string // "asc", "desc" — default "desc"
}

type StatsFilters struct {
	Year  *int
	Month *int
	Week  *int
	From  *time.Time
	To    *time.Time
}

type Stats struct {
	RideCount       int
	TotalDistanceM  float64
	TotalDurationS  int
	TotalElevationM float64
	AvgSpeedMPS     float64
	MaxSpeedMPS     float64
	AvgPowerW       *float64
	MaxPowerW       *float64
	AvgHRBPM        *float64
	MaxHRBPM        *float64
}

// RecordsFilters controls which rides are considered when computing personal records.
type RecordsFilters struct {
	Year  *int
	Month *int
	Week  *int
	From  *time.Time
	To    *time.Time
}

// PersonalRecord holds the raw value and date of a single personal best.
// A nil *PersonalRecord means no data exists for that category.
type PersonalRecord struct {
	RawValue float64   `json:"value"`
	Date     time.Time `json:"date"`
}

// Records holds all personal best categories. Nil fields mean no data.
type Records struct {
	LongestDistanceM   *PersonalRecord `json:"longest_distance_m"`
	LongestDurationS   *PersonalRecord `json:"longest_duration_s"`
	MostElevationGainM *PersonalRecord `json:"most_elevation_gain_m"`
	HighestAvgPowerW   *PersonalRecord `json:"highest_avg_power_w"`
	HighestAvgSpeedMPS *PersonalRecord `json:"highest_avg_speed_mps"`
	HighestAvgHRBPM    *PersonalRecord `json:"highest_avg_hr_bpm"`
	HighestMaxSpeedMPS *PersonalRecord `json:"highest_max_speed_mps"`
	MostCaloriesKcal   *PersonalRecord `json:"most_calories_kcal"`
	HighestAltitudeM   *PersonalRecord `json:"highest_altitude_m"`
}

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

// GPSPoint is a single GPS sample from the streams table.
type GPSPoint struct {
	ElapsedS  int
	Lat       float64
	Lon       float64
	AltitudeM *float64
}
