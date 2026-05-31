package display

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
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

func TestPrintRideList_JSON(t *testing.T) {
	var buf bytes.Buffer
	rides := []parser.Ride{
		{
			Position:    1,
			RecordedAt:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			DistanceM:   30000,
			DurationS:   3600,
			AvgSpeedMPS: 8.3,
		},
	}
	PrintRideList(&buf, rides, 1, 1, 10, true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"distance_m"`) {
		t.Errorf("expected JSON key 'distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}

func TestPrintRideDetail_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := parser.Ride{
		RecordedAt:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		DistanceM:   50000,
		DurationS:   7200,
		AvgSpeedMPS: 6.9,
	}
	PrintRideDetail(&buf, r, true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"distance_m"`) {
		t.Errorf("expected JSON key 'distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}

func TestPrintStats_JSON(t *testing.T) {
	var buf bytes.Buffer
	avgPower := 231.0
	st := store.Stats{
		RideCount:      5,
		TotalDistanceM: 100000,
		TotalDurationS: 18000,
		AvgSpeedMPS:    7.5,
		MaxSpeedMPS:    12.0,
		AvgPowerW:      &avgPower,
	}
	PrintStats(&buf, st, "all time", true, "metric")
	output := buf.String()
	// nil pointer fields still appear in JSON output as null, so all keys are always present
	for _, want := range []string{
		`"RideCount"`, `"AvgSpeedMPS"`, `"MaxSpeedMPS"`,
		`"AvgPowerW"`, `"MaxPowerW"`, `"AvgHRBPM"`, `"MaxHRBPM"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected JSON key %s in output, got:\n%s", want, output)
		}
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}

func TestPrintStats_WithPowerAndHR(t *testing.T) {
	var buf bytes.Buffer
	avgPower, maxPower := 231.0, 421.0
	avgHR, maxHR := 148.0, 183.0
	st := store.Stats{
		RideCount:       14,
		TotalDistanceM:  423700,
		TotalDurationS:  67320,
		TotalElevationM: 4210,
		AvgSpeedMPS:     7.89,
		MaxSpeedMPS:     14.47,
		AvgPowerW:       &avgPower,
		MaxPowerW:       &maxPower,
		AvgHRBPM:        &avgHR,
		MaxHRBPM:        &maxHR,
	}
	PrintStats(&buf, st, "all time", false, "metric")
	output := buf.String()
	for _, want := range []string{
		"Avg Speed", "Max Speed", "28.4 km/h", "52.1 km/h",
		"Avg Power", "231 W",
		"Max Power", "421 W",
		"Avg HR", "148 bpm",
		"Max HR", "183 bpm",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestPrintStats_NoPowerNoHR(t *testing.T) {
	var buf bytes.Buffer
	st := store.Stats{
		RideCount:      3,
		TotalDistanceM: 87200,
		TotalDurationS: 11640,
		AvgSpeedMPS:    7.5,
		MaxSpeedMPS:    12.0,
	}
	PrintStats(&buf, st, "all time", false, "metric")
	output := buf.String()
	for _, want := range []string{"Avg Speed", "Max Speed"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
	for _, absent := range []string{"Avg Power", "Max Power", "Avg HR", "Max HR"} {
		if strings.Contains(output, absent) {
			t.Errorf("expected %q to be absent from output:\n%s", absent, output)
		}
	}
}

func TestPrintStats_EmptyDB(t *testing.T) {
	var buf bytes.Buffer
	st := store.Stats{} // zero value: RideCount=0, all speeds=0, all pointers nil
	PrintStats(&buf, st, "all time", false, "metric")
	output := buf.String()
	// Speed rows always appear even when RideCount is 0
	for _, want := range []string{"Avg Speed", "Max Speed", "0.0 km/h"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
	// Power and HR rows must be absent (nil fields)
	for _, absent := range []string{"Avg Power", "Max Power", "Avg HR", "Max HR"} {
		if strings.Contains(output, absent) {
			t.Errorf("expected %q to be absent from output:\n%s", absent, output)
		}
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

func TestPrintRecords_FullTable(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	pr := func(v float64) *store.PersonalRecord {
		return &store.PersonalRecord{RawValue: v, Date: date}
	}
	recs := store.Records{
		LongestDistanceM:   pr(50000),
		LongestDurationS:   pr(7200),
		MostElevationGainM: pr(1500),
		HighestAvgPowerW:   pr(250),
		HighestAvgSpeedMPS: pr(10.0),
		HighestAvgHRBPM:    pr(155),
		HighestMaxSpeedMPS: pr(20.0),
		MostCaloriesKcal:   pr(800),
		HighestAltitudeM:   pr(1200),
	}
	PrintRecords(&buf, recs, "all time", false, "metric")
	output := buf.String()

	for _, want := range []string{
		"Longest distance", "50.0 km", "2024-06-15",
		"Longest duration", "2h 00m 00s",
		"Most elevation gain", "1500 m",
		"Highest avg power", "250 W",
		"Highest avg speed", "36.0 km/h",
		"Highest avg HR", "155 bpm",
		"Highest max speed", "72.0 km/h",
		"Most calories", "800",
		"Highest altitude", "1200 m",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestPrintRecords_PartialTable(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	pr := func(v float64) *store.PersonalRecord {
		return &store.PersonalRecord{RawValue: v, Date: date}
	}
	recs := store.Records{
		LongestDistanceM:   pr(50000),
		LongestDurationS:   pr(7200),
		MostElevationGainM: pr(1500),
		HighestAvgPowerW:   nil, // no power data
		HighestAvgSpeedMPS: pr(10.0),
		HighestAvgHRBPM:    nil, // no HR data
		HighestMaxSpeedMPS: pr(20.0),
		MostCaloriesKcal:   nil, // no calorie data
		HighestAltitudeM:   nil, // no stream altitude data
	}
	PrintRecords(&buf, recs, "all time", false, "metric")
	output := buf.String()

	if !strings.Contains(output, "Longest distance") {
		t.Errorf("expected 'Longest distance' in output, got:\n%s", output)
	}
	for _, absent := range []string{"Highest avg power", "Highest avg HR", "Most calories", "Highest altitude"} {
		if strings.Contains(output, absent) {
			t.Errorf("expected %q to be absent from output:\n%s", absent, output)
		}
	}
}

func TestPrintRecords_EmptyDB(t *testing.T) {
	var buf bytes.Buffer
	PrintRecords(&buf, store.Records{}, "all time", false, "metric")
	output := buf.String()
	if !strings.Contains(output, "No rides imported yet") {
		t.Errorf("expected empty-DB message, got:\n%s", output)
	}
}

func TestPrintRecords_EmptyFilter(t *testing.T) {
	var buf bytes.Buffer
	PrintRecords(&buf, store.Records{}, "2099", false, "metric")
	output := buf.String()
	if !strings.Contains(output, "No rides found for the selected period") {
		t.Errorf("expected period message, got:\n%s", output)
	}
}

func TestPrintRecords_JSON(t *testing.T) {
	var buf bytes.Buffer
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	recs := store.Records{
		LongestDistanceM: &store.PersonalRecord{RawValue: 50000, Date: date},
	}
	PrintRecords(&buf, recs, "all time", true, "metric")
	output := buf.String()
	if !strings.Contains(output, `"longest_distance_m"`) {
		t.Errorf("expected JSON key 'longest_distance_m', got:\n%s", output)
	}
	if !strings.Contains(output, "50000") {
		t.Errorf("expected value 50000 in JSON, got:\n%s", output)
	}
	if !strings.Contains(output, "null") {
		t.Errorf("expected null fields in JSON for absent records, got:\n%s", output)
	}
	if !strings.Contains(output, "\n  ") {
		t.Errorf("expected indented JSON (newline + 2 spaces), got:\n%s", output)
	}
}

func TestFormatStatsDelta_Positive(t *testing.T) {
	got := formatStatsDelta(100000, 80000, func(v float64) string { return FormatDistance(v, "metric") })
	// delta = 20000 m = +20.0 km; pct = 20000/80000*100 = 25%
	if !strings.Contains(got, "+20.0 km") {
		t.Errorf("expected '+20.0 km' in %q", got)
	}
	if !strings.Contains(got, "+25%") {
		t.Errorf("expected '+25%%' in %q", got)
	}
}

func TestFormatStatsDelta_Negative(t *testing.T) {
	got := formatStatsDelta(80000, 100000, func(v float64) string { return FormatDistance(v, "metric") })
	// delta = -20000 m = -20.0 km; pct = -20000/100000*100 = -20%
	if !strings.Contains(got, "-20.0 km") {
		t.Errorf("expected '-20.0 km' in %q", got)
	}
	if !strings.Contains(got, "-20%") {
		t.Errorf("expected '-20%%' in %q", got)
	}
}

func TestFormatStatsDelta_ZeroBase(t *testing.T) {
	got := formatStatsDelta(5, 0, func(v float64) string { return strconv.Itoa(int(v)) })
	// base is zero: no percentage, just "+5"
	if got != "+5" {
		t.Errorf("formatStatsDelta zero base = %q, want %q", got, "+5")
	}
}

func TestPrintStatsComparison_Table(t *testing.T) {
	st1 := store.Stats{
		RideCount:       10,
		TotalDistanceM:  100000,
		TotalDurationS:  3600,  // "1h 00m 00s"
		TotalElevationM: 1000,
	}
	st2 := store.Stats{
		RideCount:       8,
		TotalDistanceM:  80000,
		TotalDurationS:  3200, // "53m 20s"
		TotalElevationM: 800,
	}
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", false, "metric")
	out := buf.String()

	for _, want := range []string{
		"2025 vs 2024",
		"1h 00m 00s",  // st1 duration
		"53m 20s",     // st2 duration
		"100.0 km",    // st1 distance
		"80.0 km",     // st2 distance
		"+20.0 km",    // distance delta value
		"+6m 40s",     // duration delta (delta = 400s = 6m40s)
		"+200 m",      // elevation delta value
		"+25%",        // rides / distance / elevation pct
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestPrintStatsComparison_ZeroBase(t *testing.T) {
	st1 := store.Stats{
		RideCount:       5,
		TotalDistanceM:  50000,
		TotalDurationS:  1800, // "30m 00s"
		TotalElevationM: 500,
	}
	st2 := store.Stats{} // all zeros
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", false, "metric")
	out := buf.String()

	for _, want := range []string{"+5", "+50.0 km", "+30m 00s", "+500 m"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
	if strings.Contains(out, "%") {
		t.Errorf("expected no percentage when base is zero, got:\n%s", out)
	}
}

func TestPrintStreamChart_SingleField(t *testing.T) {
	power := 250
	points := []parser.Stream{
		{PowerW: &power},
		{PowerW: &power},
		{PowerW: &power},
	}
	var buf bytes.Buffer
	PrintStreamChart(&buf, [][]parser.Stream{points}, []string{"power"})
	out := buf.String()
	if !strings.Contains(out, "power") {
		t.Errorf("expected caption 'power' in output, got:\n%s", out)
	}
	if strings.Contains(out, "points") {
		t.Errorf("expected no point count in caption, got:\n%s", out)
	}
}

func TestPrintStreamChart_SingleField_NoData(t *testing.T) {
	var buf bytes.Buffer
	PrintStreamChart(&buf, [][]parser.Stream{{}}, []string{"power"})
	out := buf.String()
	if !strings.Contains(out, "No power data") {
		t.Errorf("expected no-data message, got:\n%s", out)
	}
}

func TestPrintStatsComparison_JSON(t *testing.T) {
	st1 := store.Stats{RideCount: 10, TotalDistanceM: 100000}
	st2 := store.Stats{RideCount: 8, TotalDistanceM: 80000}
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", true, "metric")

	var out struct {
		Primary struct {
			Label string      `json:"label"`
			Stats store.Stats `json:"stats"`
		} `json:"primary"`
		Compare struct {
			Label string      `json:"label"`
			Stats store.Stats `json:"stats"`
		} `json:"compare"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if out.Primary.Label != "2025" {
		t.Errorf("primary label = %q, want %q", out.Primary.Label, "2025")
	}
	if out.Compare.Label != "2024" {
		t.Errorf("compare label = %q, want %q", out.Compare.Label, "2024")
	}
	if out.Primary.Stats.RideCount != 10 {
		t.Errorf("primary ride count = %d, want 10", out.Primary.Stats.RideCount)
	}
	if out.Compare.Stats.RideCount != 8 {
		t.Errorf("compare ride count = %d, want 8", out.Compare.Stats.RideCount)
	}
}
