package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var streamFields []string

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Show a time-series chart for a ride stream",
	Args:  cobra.NoArgs,
	RunE:  runStream,
}

func init() {
	streamCmd.Flags().StringSliceVar(&streamFields, "field", nil, "field(s) to chart: power, hr, speed, cadence, altitude (repeatable or comma-separated)")
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

	fields := streamFields
	if len(fields) == 0 {
		// Default: first available field by priority power → hr → speed.
		available, err := s.AvailableFields(id)
		if err != nil {
			return err
		}
		for _, candidate := range []string{"power", "hr", "speed"} {
			for _, a := range available {
				if a == candidate {
					fields = []string{candidate}
					break
				}
			}
			if len(fields) > 0 {
				break
			}
		}
		if len(fields) == 0 {
			return fmt.Errorf("ride #%d has no stream data", currentRide.Position)
		}
	}

	for i, field := range fields {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		points, err := s.GetStreams(id, field)
		if err != nil {
			return err
		}
		if len(points) == 0 {
			available, _ := s.AvailableFields(id)
			fmt.Fprintf(os.Stderr, "No %s data for ride #%d. Available fields: %v\n", field, currentRide.Position, available)
			continue
		}
		display.PrintStreamChart(os.Stdout, [][]parser.Stream{points}, []string{field})
	}
	return nil
}
