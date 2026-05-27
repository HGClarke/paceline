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
		_ = json.NewEncoder(w).Encode(rides) // write error on stdout is unrecoverable
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(tablewriter.WithBorders(tw.Border{
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"#", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
	for _, r := range rides {
		table.Append([]string{
			strconv.FormatInt(r.Position, 10),
			r.RecordedAt.Format("2006-01-02"),
			FormatDistance(r.DistanceM, units),
			formatDuration(r.DurationS),
			FormatElevation(r.ElevationGainM, units),
			formatSpeed(r.AvgSpeedMPS, units),
		})
	}
	table.Render()
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
		_ = json.NewEncoder(w).Encode(r) // write error on stdout is unrecoverable
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{
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
	table.Bulk(rows)
	table.Render()
}

// PrintStats renders aggregated stats to w.
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool, units string) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(st) // write error on stdout is unrecoverable
		return
	}
	fmt.Fprintf(w, "Stats: %s\n\n", label)
	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Bulk([][]string{
		{"Rides", strconv.Itoa(st.RideCount)},
		{"Total Distance", FormatDistance(st.TotalDistanceM, units)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", FormatElevation(st.TotalElevationM, units)},
	})
	table.Render()
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
