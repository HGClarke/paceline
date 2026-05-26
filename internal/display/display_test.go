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
	PrintRideList(&buf, rides, 1, 1, 10, false)
	output := buf.String()
	if !strings.Contains(output, "#") {
		t.Errorf("expected '#' column header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("expected position '42' in output, got:\n%s", output)
	}
}
