package cmd

import (
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var powerCurveCmd = &cobra.Command{
	Use:   "power-curve",
	Short: "Show the mean maximal power curve for a ride",
	Args:  cobra.NoArgs,
	RunE:  runPowerCurve,
}

func runPowerCurve(_ *cobra.Command, _ []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	curve, err := s.GetPowerCurve(currentRideID)
	if err != nil {
		return err
	}

	display.PrintPowerCurve(os.Stdout, curve, currentRide.Position, currentRide.RecordedAt)
	return nil
}
