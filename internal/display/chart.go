package display

import (
	"fmt"
	"io"

	"github.com/guptarohit/asciigraph"
	"github.com/HGClarke/paceline/internal/parser"
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
