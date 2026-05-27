package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete ride data",
}

var deleteRideCmd = &cobra.Command{
	Use:   "ride <position>",
	Short: "Delete a specific ride and its stream data",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteRide,
}

var deleteAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Delete all rides and stream data",
	Args:  cobra.NoArgs,
	RunE:  runDeleteAll,
}

func init() {
	deleteRideCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation prompt")
	deleteAllCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation prompt")
	deleteCmd.AddCommand(deleteRideCmd)
	deleteCmd.AddCommand(deleteAllCmd)
	rootCmd.AddCommand(deleteCmd)
}

func runDeleteRide(cmd *cobra.Command, args []string) error {
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

	prompt := fmt.Sprintf("Delete ride #%d (%s, %s)?",
		ride.Position, ride.RecordedAt.Format("2006-01-02"), display.FormatDistance(ride.DistanceM, cfg.Units))

	if !confirm(prompt, deleteForce) {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := s.DeleteRide(ride.ID); err != nil {
		return err
	}
	fmt.Printf("Deleted ride #%d.\n", ride.Position)
	return nil
}

func runDeleteAll(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	stats, err := s.GetStats(store.StatsFilters{})
	if err != nil {
		return err
	}
	if stats.RideCount == 0 {
		fmt.Println("No rides to delete.")
		return nil
	}

	prompt := fmt.Sprintf("Delete all %d rides? This cannot be undone.", stats.RideCount)
	if !confirm(prompt, deleteForce) {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := s.DeleteAll(); err != nil {
		return err
	}
	fmt.Printf("Deleted all %d rides.\n", stats.RideCount)
	return nil
}

// confirm prints a [y/N] prompt and returns true only if the user types y or Y.
// If force is true, it skips the prompt and returns true immediately.
func confirm(prompt string, force bool) bool {
	if force {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y"
	}
	return false
}
