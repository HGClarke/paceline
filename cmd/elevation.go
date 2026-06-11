package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var elevationCmd = &cobra.Command{
	Use:   "elevation",
	Short: "Show ASCII elevation profile for a ride",
	Args:  cobra.NoArgs,
	RunE:  runElevation,
}

func runElevation(_ *cobra.Command, _ []string) error {
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

	display.PrintElevationProfile(os.Stdout, pts, cfg.Units, currentRide.Position, currentRide.RecordedAt)
	return nil
}
