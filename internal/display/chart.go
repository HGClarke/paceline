package display

import (
	"fmt"
	"io"

	"github.com/guptarohit/asciigraph"
	"github.com/HGClarke/paceline/internal/parser"
)

// PrintStreamChart renders a terminal line chart for the given stream points.
// field is used only for the chart caption.
func PrintStreamChart(w io.Writer, points []parser.Stream, field string) {
	data := extractFieldValues(points, field)
	if len(data) == 0 {
		fmt.Fprintf(w, "No %s data to display.\n", field)
		return
	}

	caption := fmt.Sprintf("%s over time (%d points)", field, len(data))
	chart := asciigraph.Plot(data, asciigraph.Height(15), asciigraph.Width(80), asciigraph.Caption(caption))
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
				vals = append(vals, *p.SpeedMPS*3.6) // convert to km/h
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
