package display

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/guptarohit/asciigraph"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

var fieldColor = map[string]asciigraph.AnsiColor{
	"power":    asciigraph.Red,
	"hr":       asciigraph.Blue,
	"speed":    asciigraph.Green,
	"cadence":  asciigraph.Yellow,
	"altitude": asciigraph.Cyan,
}

// PrintStreamChart renders a terminal line chart for one or more stream fields.
// allSeries[i] contains the stream points for fields[i].
// Single field: uses asciigraph.Plot with color. Multiple fields: uses PlotMany with legends.
func PrintStreamChart(w io.Writer, allSeries [][]parser.Stream, fields []string) {
	if len(fields) == 1 {
		data := extractFieldValues(allSeries[0], fields[0])
		if len(data) == 0 {
			fmt.Fprintf(w, "No %s data to display.\n", fields[0])
			return
		}
		chart := asciigraph.Plot(data,
			asciigraph.Height(15),
			asciigraph.Width(80),
			asciigraph.Caption(fields[0]),
			asciigraph.SeriesColors(fieldColor[fields[0]]),
		)
		fmt.Fprintln(w, chart)
		return
	}

	seriesData := make([][]float64, 0, len(fields))
	colors := make([]asciigraph.AnsiColor, 0, len(fields))
	for i, field := range fields {
		seriesData = append(seriesData, extractFieldValues(allSeries[i], field))
		colors = append(colors, fieldColor[field])
	}
	chart := asciigraph.PlotMany(seriesData,
		asciigraph.Height(15),
		asciigraph.Width(80),
		asciigraph.Caption("stream overlay"),
		asciigraph.SeriesLegends(fields...),
		asciigraph.SeriesColors(colors...),
	)
	fmt.Fprintln(w, chart)
}

// PrintPowerCurve renders the header, table, and chart for a ride's mean maximal power curve.
func PrintPowerCurve(w io.Writer, curve store.PowerCurve, ridePos int64, date time.Time) {
	fmt.Fprintf(w, "Power Curve — Ride %d (%s)\n\n", ridePos, date.Format("2006-01-02"))

	if len(curve.Table) == 0 {
		fmt.Fprintln(w, "No power data available for this ride.")
		return
	}

	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header([]string{"Duration", "Power"})
	for _, pt := range curve.Table {
		_ = table.Append([]string{formatMMPDuration(pt.DurationS), fmt.Sprintf("%d W", pt.PowerW)})
	}
	_ = table.Render()

	fmt.Fprintln(w)
	PrintPowerCurveChart(w, curve)
}

// formatMMPDuration formats a canonical table duration with human-readable labels.
func formatMMPDuration(s int) string {
	switch s {
	case 5:
		return "5s"
	case 30:
		return "30s"
	case 60:
		return "1 min"
	case 300:
		return "5 min"
	case 600:
		return "10 min"
	case 1200:
		return "20 min"
	case 3600:
		return "60 min"
	default:
		if s < 60 {
			return fmt.Sprintf("%ds", s)
		}
		if s < 3600 {
			return fmt.Sprintf("%d min", s/60)
		}
		return fmt.Sprintf("%dh", s/3600)
	}
}

// PrintPowerCurveChart renders the power curve ASCII chart followed by approximate X-axis labels.
func PrintPowerCurveChart(w io.Writer, curve store.PowerCurve) {
	if len(curve.Chart) == 0 {
		return
	}
	data := make([]float64, len(curve.Chart))
	for i, pt := range curve.Chart {
		data[i] = float64(pt.PowerW)
	}
	chart := asciigraph.Plot(data,
		asciigraph.Height(15),
		asciigraph.Width(80),
		asciigraph.Caption("power curve"),
		asciigraph.SeriesColors(asciigraph.Red),
	)
	fmt.Fprintln(w, chart)
	fmt.Fprintln(w, powerCurveXLabels(curve.Chart))
}

// powerCurveXLabels builds an approximate X-axis label line for the power curve chart.
// The chart width is 80 columns; the Y-axis gutter is ~9 chars wide, leaving ~71 for data.
func powerCurveXLabels(pts []store.PowerCurvePoint) string {
	const chartWidth = 71
	n := len(pts)
	if n == 0 {
		return ""
	}

	tickDurations := []int{5, 30, 60, 300, 600, 1200, 3600}
	maxDuration := pts[n-1].DurationS

	buf := make([]byte, chartWidth+10)
	for i := range buf {
		buf[i] = ' '
	}

	for _, d := range tickDurations {
		if d > maxDuration {
			break
		}
		label := formatPCDuration(d)
		pos := durationToPos(d, pts[0].DurationS, maxDuration, chartWidth)
		if pos+len(label) > len(buf) {
			continue
		}
		copy(buf[pos:], label)
	}

	return "         " + strings.TrimRight(string(buf), " ")
}

// durationToPos maps a duration to a character position in the chart data area (log scale).
func durationToPos(d, minD, maxD, width int) int {
	if maxD <= minD {
		return 0
	}
	logMin := math.Log(float64(minD))
	logMax := math.Log(float64(maxD))
	logD := math.Log(float64(d))
	frac := (logD - logMin) / (logMax - logMin)
	pos := int(frac * float64(width-1))
	if pos < 0 {
		pos = 0
	}
	if pos >= width {
		pos = width - 1
	}
	return pos
}

// formatPCDuration formats a duration in seconds for X-axis labels.
func formatPCDuration(s int) string {
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	default:
		return fmt.Sprintf("%dh", s/3600)
	}
}

func extractFieldValues(points []parser.Stream, field string) []float64 {
	var vals []float64
	for _, p := range points {
		switch field {
		case "power":
			if p.PowerW != nil {
				vals = append(vals, float64(*p.PowerW))
			}
		case "hr":
			if p.HRBPM != nil {
				vals = append(vals, float64(*p.HRBPM))
			}
		case "speed":
			if p.SpeedMPS != nil {
				vals = append(vals, *p.SpeedMPS*3.6)
			}
		case "cadence":
			if p.CadenceRPM != nil {
				vals = append(vals, float64(*p.CadenceRPM))
			}
		case "altitude":
			if p.AltitudeM != nil {
				vals = append(vals, *p.AltitudeM)
			}
		}
	}
	return vals
}
