package display

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/HGClarke/paceline/internal/store"
	"github.com/guptarohit/asciigraph"
)

// PrintElevationProfile renders an ASCII elevation chart (altitude vs distance) to w.
// pts must come from store.GetGPSPoints; points without AltitudeM are silently skipped.
func PrintElevationProfile(w io.Writer, pts []store.GPSPoint, units string, ridePos int64, date time.Time) {
	fmt.Fprintf(w, "Elevation Profile — Ride %d (%s)\n\n", ridePos, date.Format("2006-01-02"))

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

	cumDist := 0.0
	displayAlt := make([]float64, len(altPts))
	altUnit := "m"
	for i, p := range altPts {
		if i > 0 {
			cumDist += haversineM(altPts[i-1].Lat, altPts[i-1].Lon, p.Lat, p.Lon)
		}
		if units == "imperial" {
			displayAlt[i] = *p.AltitudeM * 3.28084
			altUnit = "ft"
		} else {
			displayAlt[i] = *p.AltitudeM
		}
	}

	caption := fmt.Sprintf("elevation (%s)", altUnit)
	chart := asciigraph.Plot(displayAlt,
		asciigraph.Height(15),
		asciigraph.Width(80),
		asciigraph.Caption(caption),
		asciigraph.SeriesColors(asciigraph.Cyan),
	)
	fmt.Fprintln(w, chart)
	fmt.Fprintln(w, elevXLabels(cumDist, len(altPts), units))
}

// haversineM returns the great-circle distance in metres between two lat/lon points.
func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// elevXLabels builds a distance X-axis label line for the elevation chart.
// The chart is 80 cols wide; the Y-axis gutter is ~9 chars, leaving ~71 for data.
func elevXLabels(totalDistM float64, nPts int, units string) string {
	const chartWidth = 71
	if nPts <= 0 || totalDistM == 0 {
		return ""
	}

	buf := make([]byte, chartWidth+10)
	for i := range buf {
		buf[i] = ' '
	}

	for _, f := range []float64{0, 0.25, 0.5, 0.75, 1.0} {
		label := formatElevDist(totalDistM*f, units)
		pos := int(f * float64(chartWidth-1))
		if pos+len(label) > len(buf) {
			pos = len(buf) - len(label)
		}
		copy(buf[pos:], label)
	}

	return "         " + strings.TrimRight(string(buf), " ")
}

// formatElevDist formats a distance in metres for the elevation X-axis.
func formatElevDist(m float64, units string) string {
	if units == "imperial" {
		return fmt.Sprintf("%.1fmi", m/1609.344)
	}
	return fmt.Sprintf("%.1fkm", m/1000)
}
