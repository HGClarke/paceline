package display

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintRideList renders a table of rides to w. If jsonOut is true, emits JSON instead.
func PrintRideList(w io.Writer, rides []parser.Ride, total, page, limit int, jsonOut bool) {
	if jsonOut {
		_ = json.NewEncoder(w).Encode(rides) // write error on stdout is unrecoverable
		return
	}
	table := tablewriter.NewWriter(w)
	table.Options(tablewriter.WithBorders(tw.Border{
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"ID", "Date", "Distance", "Duration", "Elevation", "Avg Speed"})
	for _, r := range rides {
		table.Append([]string{
			strconv.FormatInt(r.ID, 10),
			r.RecordedAt.Format("2006-01-02"),
			fmt.Sprintf("%.1f km", r.DistanceM/1000),
			formatDuration(r.DurationS),
			fmt.Sprintf("%.0f m", r.ElevationGainM),
			fmt.Sprintf("%.1f km/h", r.AvgSpeedMPS*3.6),
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
func PrintRideDetail(w io.Writer, r parser.Ride, jsonOut bool) {
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
		{"Date", r.RecordedAt.Format(time.RFC1123)},
		{"Distance", fmt.Sprintf("%.2f km", r.DistanceM/1000)},
		{"Duration", formatDuration(r.DurationS)},
		{"Elevation Gain", fmt.Sprintf("%.0f m", r.ElevationGainM)},
		{"Avg Speed", fmt.Sprintf("%.1f km/h", r.AvgSpeedMPS*3.6)},
		{"Max Speed", fmt.Sprintf("%.1f km/h", r.MaxSpeedMPS*3.6)},
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
func PrintStats(w io.Writer, st store.Stats, label string, jsonOut bool) {
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
		{"Total Distance", fmt.Sprintf("%.1f km", st.TotalDistanceM/1000)},
		{"Total Duration", formatDuration(st.TotalDurationS)},
		{"Total Elevation", fmt.Sprintf("%.0f m", st.TotalElevationM)},
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
