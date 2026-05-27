package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "0m 00s"},
		{30, "0m 30s"},
		{60, "1m 00s"},
		{90, "1m 30s"},
		{3599, "59m 59s"},
		{3600, "1h 00m 00s"},
		{3661, "1h 01m 01s"},
		{7384, "2h 03m 04s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestExtractFieldValues(t *testing.T) {
	power := 250
	hr := 160
	speed := 10.0 // m/s
	cadence := 90
	alt := 100.0

	points := []parser.Stream{
		{PowerW: &power, HRBPM: &hr, SpeedMPS: &speed, CadenceRPM: &cadence, AltitudeM: &alt},
		{PowerW: nil, HRBPM: nil, SpeedMPS: nil, CadenceRPM: nil, AltitudeM: nil},
	}

	tests := []struct {
		field string
		want  []float64
	}{
		{"power", []float64{250}},
		{"hr", []float64{160}},
		{"speed", []float64{36.0}}, // 10 m/s * 3.6
		{"cadence", []float64{90}},
		{"altitude", []float64{100}},
		{"unknown", nil},
	}

	for _, tt := range tests {
		got := extractFieldValues(points, tt.field)
		if len(got) != len(tt.want) {
			t.Errorf("extractFieldValues field=%q: got %v, want %v", tt.field, got, tt.want)
			continue
		}
		for i, v := range got {
			if v != tt.want[i] {
				t.Errorf("extractFieldValues field=%q [%d]: got %v, want %v", tt.field, i, v, tt.want[i])
			}
		}
	}
}

func TestPrintRideList_ShowsPositionColumn(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:       42,
			RecordedAt:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:      30000,
			DurationS:      3600,
			ElevationGainM: 500,
			AvgSpeedMPS:    8.3,
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, false, "metric")
	output := buf.String()
	if !strings.Contains(output, "#") {
		t.Errorf("expected '#' column header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("expected position '42' in output, got:\n%s", output)
	}
}

func TestPrintRideList_Imperial(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:       1,
			RecordedAt:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:      16093.44, // exactly 10 miles
			DurationS:      3600,
			ElevationGainM: 304.8, // exactly 1000 ft
			AvgSpeedMPS:    4.4704, // exactly 10 mph
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, false, "imperial")
	output := buf.String()
	if !strings.Contains(output, "mi") {
		t.Errorf("expected miles unit in output, got:\n%s", output)
	}
	if !strings.Contains(output, "mph") {
		t.Errorf("expected mph unit in output, got:\n%s", output)
	}
	if !strings.Contains(output, "ft") {
		t.Errorf("expected ft unit in output, got:\n%s", output)
	}
	if strings.Contains(output, " km") {
		t.Errorf("expected no km in imperial output, got:\n%s", output)
	}
}

func TestFormatDistance_Metric(t *testing.T) {
	tests := []struct {
		m    float64
		want string
	}{
		{1000, "1.0 km"},
		{48280, "48.3 km"},
		{0, "0.0 km"},
	}
	for _, tt := range tests {
		got := FormatDistance(tt.m, "metric")
		if got != tt.want {
			t.Errorf("FormatDistance(%.0f, metric) = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFormatDistance_Imperial(t *testing.T) {
	tests := []struct {
		m    float64
		want string
	}{
		{1609.344, "1.0 mi"},
		{80467.2, "50.0 mi"},
	}
	for _, tt := range tests {
		got := FormatDistance(tt.m, "imperial")
		if got != tt.want {
			t.Errorf("FormatDistance(%.3f, imperial) = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestFormatSpeed_Metric(t *testing.T) {
	// 10 m/s = 36 km/h
	got := formatSpeed(10.0, "metric")
	if got != "36.0 km/h" {
		t.Errorf("formatSpeed(10, metric) = %q, want %q", got, "36.0 km/h")
	}
}

func TestFormatSpeed_Imperial(t *testing.T) {
	// 1 m/s = 2.23694 mph → round to 1dp = "2.2 mph"
	got := formatSpeed(1.0, "imperial")
	if got != "2.2 mph" {
		t.Errorf("formatSpeed(1, imperial) = %q, want %q", got, "2.2 mph")
	}
}

func TestFormatElevation_Metric(t *testing.T) {
	got := FormatElevation(1420.0, "metric")
	if got != "1420 m" {
		t.Errorf("FormatElevation(1420, metric) = %q, want %q", got, "1420 m")
	}
}

func TestFormatElevation_Imperial(t *testing.T) {
	// 1 m = 3.28084 ft; 100 m = 328 ft
	got := FormatElevation(100.0, "imperial")
	if got != "328 ft" {
		t.Errorf("FormatElevation(100, imperial) = %q, want %q", got, "328 ft")
	}
}
