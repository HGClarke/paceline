package display

import "fmt"

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
