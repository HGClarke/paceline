package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var (
	routeWidth  int
	routeHeight int
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
	return nil
}
