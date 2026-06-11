package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var (
	routeWidth  int
	routeHeight int
	routeOpen   bool
)

var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Render the GPS route as a Braille map in the terminal",
	Args:  cobra.NoArgs,
	RunE:  runRoute,
}

func init() {
	routeCmd.Flags().IntVar(&routeWidth, "width", 78, "character width of the map")
	routeCmd.Flags().IntVar(&routeHeight, "height", 28, "character height of the map")
	routeCmd.Flags().BoolVar(&routeOpen, "open", false, "open route in browser on OpenStreetMap")
}

func runRoute(_ *cobra.Command, _ []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	pts, err := s.GetGPSPoints(currentRideID)
	if err != nil {
		return err
	}

	if len(pts) == 0 {
		fmt.Fprintf(os.Stderr, "Ride #%d has no GPS data.\n", currentRide.Position)
		return nil
	}

	display.PrintRoute(os.Stdout, pts, routeWidth, routeHeight, currentRide.Filename)

	if routeOpen {
		url := buildOSMURL(pts)
		fmt.Fprintf(os.Stdout, "  Opening %s\n", url)
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not open browser: %v\n", err)
		}
	}
	return nil
}

// buildOSMURL returns an OpenStreetMap bounding-box URL for the given GPS points.
// Returns "" if pts is empty.
func buildOSMURL(pts []store.GPSPoint) string {
	if len(pts) == 0 {
		return ""
	}
	minLat, maxLat, minLon, maxLon := display.GPSBounds(pts)
	return fmt.Sprintf("https://www.openstreetmap.org/?bbox=%.6f,%.6f,%.6f,%.6f",
		minLon, minLat, maxLon, maxLat)
}

// openBrowser opens url in the system default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
