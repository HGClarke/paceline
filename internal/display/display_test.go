package display

import (
	"testing"

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
