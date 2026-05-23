package cmd

import (
	"fmt"
	"os"

	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var streamField string

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Show a time-series chart for a ride stream",
	Args:  cobra.NoArgs,
	RunE:  runStream,
}

func init() {
	streamCmd.Flags().StringVar(&streamField, "field", "", "field to chart: power, hr, speed, cadence, altitude")
}

func runStream(cmd *cobra.Command, args []string) error {
	id := currentRideID

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	field := streamField
	if field == "" {
		available, err := s.AvailableFields(id)
		if err != nil {
			return err
		}
		for _, candidate := range []string{"power", "hr", "speed"} {
			for _, a := range available {
				if a == candidate {
					field = candidate
					break
				}
			}
			if field != "" {
				break
			}
		}
		if field == "" {
			return fmt.Errorf("ride %d has no stream data", id)
		}
	}

	points, err := s.GetStreams(id, field)
	if err != nil {
		return err
	}

	if len(points) == 0 {
		available, _ := s.AvailableFields(id)
		fmt.Fprintf(os.Stderr, "No %s data for ride %d. Available fields: %v\n", field, id, available)
		return nil
	}

	display.PrintStreamChart(os.Stdout, points, field)
	return nil
}
