package display

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintRideList renders a table of rides to w. If jsonOut is true, emits JSON instead.
func PrintRideList(w io.Writer, rides []parser.Ride, total, page, limit int, jsonOut bool, units string) {
	if jsonOut {
		b, _ := json.MarshalIndent(rides, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"#", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
	for _, r := range rides {
		_ = table.Append([]string{ // write errors are unrecoverable; discard return value
			strconv.FormatInt(r.Position, 10),
			r.RecordedAt.Format("2006-01-02"),
			FormatDistance(r.DistanceM, units),
			formatDuration(r.DurationS),
			FormatElevation(r.ElevationGainM, units),
			formatSpeed(r.AvgSpeedMPS, units),
		})
	}
	_ = table.Render() // write errors are unrecoverable; discard return value
	if limit > 0 {
		pages := int(math.Ceil(float64(total) / float64(limit)))
		if pages > 1 {
			fmt.Fprintf(w, "\nPage %d of %d  —  run with --page=%d for next\n", page, pages, page+1)
		}
	}
}

// PrintRideDetail renders a single ride's full summary to w.
func PrintRideDetail(w io.Writer, r parser.Ride, jsonOut bool, units string) {
	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	rows := [][]string{
		{"Date", r.RecordedAt.Format("2006-01-02")},
		{"Distance", FormatDistance(r.DistanceM, units)},
		{"Duration", formatDuration(r.DurationS)},
		{"Elevation Gain", FormatElevation(r.ElevationGainM, units)},
		{"Avg Speed", formatSpeed(r.AvgSpeedMPS, units)},
		{"Max Speed", formatSpeed(r.MaxSpeedMPS, units)},
		{"Format", r.SourceFormat},
	}
	if r.AvgHRBPM != nil {
		rows = append(rows, []string{"Avg HR", fmt.Sprintf("%d bpm", *r.AvgHRBPM)})
	}
	if r.MaxHRBPM != nil {
		rows = append(rows, []string{"Max HR", fmt.Sprintf("%d bpm", *r.MaxHRBPM)})
	}
	if r.AvgPowerW != nil {
		rows = append(rows, []string{"Avg Power", fmt.Sprintf("%d W", *r.AvgPowerW)})
	}
	if r.MaxPowerW != nil {
		rows = append(rows, []string{"Max Power", fmt.Sprintf("%d W", *r.MaxPowerW)})
	}
	if r.Calories != nil {
		rows = append(rows, []string{"Calories", strconv.Itoa(*r.Calories)})
	}
	_ = table.Bulk(rows) // write errors are unrecoverable; discard return value
	_ = table.Render()   // write errors are unrecoverable; discard return value
}

// PrintStats renders aggregated stats to w.
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool, units string) {
	if jsonOut {
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Fprintf(w, "Stats: %s\n\n", label)
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	rows := [][]string{
		{"Rides", strconv.Itoa(st.RideCount)},
		{"Total Distance", FormatDistance(st.TotalDistanceM, units)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", FormatElevation(st.TotalElevationM, units)},
		{"Avg Speed", formatSpeed(st.AvgSpeedMPS, units)},
		{"Max Speed", formatSpeed(st.MaxSpeedMPS, units)},
	}
	if st.AvgHRBPM != nil {
		rows = append(rows, []string{"Avg HR", fmt.Sprintf("%d bpm", int(math.Round(*st.AvgHRBPM)))})
	}
	if st.MaxHRBPM != nil {
		rows = append(rows, []string{"Max HR", fmt.Sprintf("%d bpm", int(math.Round(*st.MaxHRBPM)))})
	}
	if st.AvgPowerW != nil {
		rows = append(rows, []string{"Avg Power", fmt.Sprintf("%d W", int(math.Round(*st.AvgPowerW)))})
	}
	if st.MaxPowerW != nil {
		rows = append(rows, []string{"Max Power", fmt.Sprintf("%d W", int(math.Round(*st.MaxPowerW)))})
	}
	_ = table.Bulk(rows) // write errors are unrecoverable; discard return value
	_ = table.Render()   // write errors are unrecoverable; discard return value
}

// PrintRecords renders personal records to w. Nil fields in recs are omitted entirely.
// If all fields are nil, a message is printed instead of an empty table.
// If jsonOut is true, recs is serialised as JSON (nil fields appear as null).
func PrintRecords(w io.Writer, recs store.Records, label string, jsonOut bool, units string) {
	if jsonOut {
		b, _ := json.MarshalIndent(recs, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}

	fmt.Fprintf(w, "Personal Records: %s\n\n", label)

	type recordRow struct {
		name   string
		pr     *store.PersonalRecord
		format func(v float64) string
	}

	rows := []recordRow{
		{
			name:   "Longest distance",
			pr:     recs.LongestDistanceM,
			format: func(v float64) string { return FormatDistance(v, units) },
		},
		{
			name:   "Longest duration",
			pr:     recs.LongestDurationS,
			format: func(v float64) string { return formatDuration(int(v)) },
		},
		{
			name:   "Most elevation gain",
			pr:     recs.MostElevationGainM,
			format: func(v float64) string { return FormatElevation(v, units) },
		},
		{
			name:   "Highest avg power",
			pr:     recs.HighestAvgPowerW,
			format: func(v float64) string { return fmt.Sprintf("%d W", int(v)) },
		},
		{
			name:   "Highest avg speed",
			pr:     recs.HighestAvgSpeedMPS,
			format: func(v float64) string { return formatSpeed(v, units) },
		},
		{
			name:   "Highest avg HR",
			pr:     recs.HighestAvgHRBPM,
			format: func(v float64) string { return fmt.Sprintf("%d bpm", int(v)) },
		},
		{
			name:   "Highest max speed",
			pr:     recs.HighestMaxSpeedMPS,
			format: func(v float64) string { return formatSpeed(v, units) },
		},
		{
			name:   "Most calories",
			pr:     recs.MostCaloriesKcal,
			format: func(v float64) string { return strconv.Itoa(int(v)) },
		},
		{
			name:   "Highest altitude",
			pr:     recs.HighestAltitudeM,
			format: func(v float64) string { return FormatElevation(v, units) },
		},
	}

	tableRows := make([][]string, 0, len(rows))
	for _, r := range rows {
		if r.pr == nil {
			continue
		}
		tableRows = append(tableRows, []string{
			r.name,
			r.format(r.pr.RawValue),
			r.pr.Date.Format("2006-01-02"),
		})
	}

	if len(tableRows) == 0 {
		if label == "all time" {
			fmt.Fprintln(w, "No rides imported yet — run 'paceline import <file>' to get started.")
		} else {
			fmt.Fprintln(w, "No rides found for the selected period.")
		}
		return
	}

	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header([]string{"Record", "Value", "Date"})
	for _, r := range tableRows {
		_ = table.Append(r) // write errors are unrecoverable; discard return value
	}
	_ = table.Render() // write errors are unrecoverable; discard return value
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	return fmt.Sprintf("%dm %02ds", m, s)
}

// FormatDistance formats meters as "X.X km" (metric) or "X.X mi" (imperial).
func FormatDistance(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1f mi", m/1609.344)
	}
	return fmt.Sprintf("%.1f km", m/1000)
}

// formatSpeed formats m/s as "X.X km/h" (metric) or "X.X mph" (imperial).
func formatSpeed(mps float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1f mph", mps*2.23694)
	}
	return fmt.Sprintf("%.1f km/h", mps*3.6)
}

// FormatElevation formats meters as "X m" (metric) or "X ft" (imperial).
func FormatElevation(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.0f ft", m*3.28084)
	}
	return fmt.Sprintf("%.0f m", m)
}
