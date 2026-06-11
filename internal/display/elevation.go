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
)

// PrintElevationProfile renders an ASCII elevation chart (altitude vs distance) to w.
// pts must come from store.GetGPSPoints; points without AltitudeM are silently skipped.
func PrintElevationProfile(w io.Writer, pts []store.GPSPoint, units string, ridePos int64, date time.Time) {
	var altPts []store.GPSPoint
	for _, p := range pts {
		if p.AltitudeM != nil {
			altPts = append(altPts, p)
		}
	}
	if len(altPts) == 0 {
		fmt.Fprintln(w, "No altitude data available for this ride.")
		return
	}

	fmt.Fprintf(w, "Elevation Profile — Ride %d (%s)\n\n", ridePos, date.Format("2006-01-02"))

	altUnit, factor := "m", 1.0
	if units == "imperial" {
		altUnit, factor = "ft", 3.28084
	}

	cumDists := make([]float64, len(altPts))
	displayAlt := make([]float64, len(altPts))
	for i, p := range altPts {
		if i > 0 {
			cumDists[i] = cumDists[i-1] + parser.HaversineM(altPts[i-1].Lat, altPts[i-1].Lon, p.Lat, p.Lon)
		}
		displayAlt[i] = *p.AltitudeM * factor
	}

	caption := fmt.Sprintf("elevation (%s)", altUnit)
	chart := asciigraph.Plot(displayAlt,
		asciigraph.Height(15),
		asciigraph.Width(asciigraphWidth),
		asciigraph.Caption(caption),
		asciigraph.SeriesColors(asciigraph.Cyan),
	)
	fmt.Fprintln(w, chart)
	fmt.Fprintln(w, elevXLabels(cumDists, units))
}

// elevXLabels builds a distance X-axis label line for the elevation chart.
// Labels at 0%, 25%, 50%, 75%, and 100% of the chart width show the actual
// cumulative distance at those data point indices, matching chart position to real distance.
func elevXLabels(cumDists []float64, units string) string {
	n := len(cumDists)
	if n == 0 || cumDists[n-1] == 0 {
		return ""
	}

	buf := make([]byte, asciigraphDataWidth+10)
	for i := range buf {
		buf[i] = ' '
	}

	for _, f := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		idx := int(math.Round(f * float64(n-1)))
		label := formatElevDist(cumDists[idx], units)
		pos := int(math.Round(f * float64(asciigraphDataWidth-1)))
		if pos+len(label) > len(buf) {
			pos = len(buf) - len(label)
		}
		copy(buf[pos:], label)
	}

	return asciigraphGutter + strings.TrimRight(string(buf), " ")
}

// formatElevDist formats a distance in metres for the elevation X-axis.
func formatElevDist(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1fmi", m/1609.344)
	}
	return fmt.Sprintf("%.1fkm", m/1000)
}
