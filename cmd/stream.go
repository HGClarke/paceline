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
var streamOverlay bool

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Show a time-series chart for a ride stream",
	Args:  cobra.NoArgs,
	RunE:  runStream,
}

func init() {
	streamCmd.Flags().StringSliceVar(&streamFields, "field", nil, "field(s) to chart: power, hr, speed, cadence, altitude (repeatable or comma-separated)")
	streamCmd.Flags().BoolVar(&streamOverlay, "overlay", false, "render all fields on a single overlaid chart")
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

	if streamOverlay && len(fields) < 2 {
		fmt.Fprintln(os.Stderr, "warning: --overlay requires at least 2 --field values; rendering single chart")
	}

	noData := func(field string) {
		msg := fmt.Sprintf("No %s data for ride #%d", field, currentRide.Position)
		if available, err := s.AvailableFields(id); err == nil {
			msg += fmt.Sprintf(". Available fields: %v", available)
		}
		fmt.Fprintln(os.Stderr, msg)
	}

	if streamOverlay {
		allSeries := make([][]parser.Stream, 0, len(fields))
		validFields := make([]string, 0, len(fields))
		for _, field := range fields {
			points, err := s.GetStreams(id, field)
			if err != nil {
				return err
			}
			if len(points) == 0 {
				noData(field)
				continue
			}
			allSeries = append(allSeries, points)
			validFields = append(validFields, field)
		}
		if len(validFields) == 0 {
			return fmt.Errorf("ride #%d has no stream data for requested fields", currentRide.Position)
		}
		display.PrintStreamChart(os.Stdout, allSeries, validFields)
		return nil
	}

	chartPrinted := false
	for _, field := range fields {
		points, err := s.GetStreams(id, field)
		if err != nil {
			return err
		}
		if len(points) == 0 {
			noData(field)
			continue
		}
		if chartPrinted {
			fmt.Fprintln(os.Stdout)
		}
		display.PrintStreamChart(os.Stdout, [][]parser.Stream{points}, []string{field})
		chartPrinted = true
	}
	return nil
}
