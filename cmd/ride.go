package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

// currentRideID is set by rideCmd's PersistentPreRunE so that the stream
// subcommand can read it without re-parsing the positional argument.
var currentRideID int64

var rideCmd = &cobra.Command{
	Use:   "ride <position>",
	Short: "Show summary stats for a specific ride",
	Args:  cobra.ArbitraryArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("requires a ride position")
		}
		pos, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid position %q: must be a number", args[0])
		}
		dbPath, err := store.DefaultPath()
		if err != nil {
			return err
		}
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer s.Close()
		ride, err := s.GetRideByPosition(pos)
		if err != nil {
			return err
		}
		currentRideID = ride.ID
		return nil
	},
	RunE: runRide,
}

func init() {
	rootCmd.AddCommand(rideCmd)
	rideCmd.AddCommand(streamCmd)
	// Mirror --field on rideCmd so cobra parses it when `ride <id> stream --field=...`
	// is invoked (cobra routes through rideCmd since <id> is numeric, not a subcommand name).
	// This flag MUST stay in sync with the streamCmd --field flag in stream.go.
	rideCmd.Flags().StringSliceVar(&streamFields, "field", nil, "field(s) to chart: power, hr, speed, cadence, altitude")
	_ = rideCmd.Flags().MarkHidden("field")
}

func runRide(cmd *cobra.Command, args []string) error {
	// If the second argument is a known subcommand name, delegate to it.
	// This handles `ride <id> stream [flags]` since cobra cannot route
	// through a numeric first arg to find the subcommand.
	if len(args) >= 2 {
		subName := args[1]
		for _, sub := range cmd.Commands() {
			if sub.Name() == subName {
				// Parse remaining args as flags for the subcommand, then run it.
				if err := sub.ParseFlags(args[2:]); err != nil {
					return err
				}
				remaining := sub.Flags().Args()
				if sub.Args != nil {
					if err := sub.Args(sub, remaining); err != nil {
						return err
					}
				}
				return sub.RunE(sub, remaining)
			}
		}
	}

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	ride, err := s.GetRide(currentRideID)
	if err != nil {
		return fmt.Errorf("ride %d not found", currentRideID)
	}

	display.PrintRideDetail(os.Stdout, ride, jsonOutput)
	return nil
}
