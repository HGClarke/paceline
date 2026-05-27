package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/parser"
	"github.com/hollandclarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var ridesCmd = &cobra.Command{
	Use:   "rides",
	Short: "List rides",
	RunE:  runRides,
}

var (
	ridesYear  int
	ridesMonth int
	ridesDate  string
	ridesPage  int
	ridesLimit int
)

func init() {
	rootCmd.AddCommand(ridesCmd)
	ridesCmd.Flags().IntVar(&ridesYear, "year", 0, "filter by year (e.g. 2024)")
	ridesCmd.Flags().IntVar(&ridesMonth, "month", 0, "filter by month (1-12)")
	ridesCmd.Flags().StringVar(&ridesDate, "date", "", "filter by date (YYYY-MM-DD)")
	ridesCmd.Flags().IntVar(&ridesPage, "page", 1, "page number")
	ridesCmd.Flags().IntVar(&ridesLimit, "limit", 10, "results per page")
}

func runRides(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	f := store.RideFilters{Page: ridesPage, Limit: ridesLimit}

	if ridesYear != 0 {
		f.Year = &ridesYear
	}
	if ridesMonth != 0 {
		if f.Year == nil {
			y := time.Now().Year()
			f.Year = &y
		}
		f.Month = &ridesMonth
	}
	if ridesDate != "" {
		t, err := time.Parse("2006-01-02", ridesDate)
		if err != nil {
			return fmt.Errorf("invalid date %q: use YYYY-MM-DD", ridesDate)
		}
		f.Date = &t
	}

	loadPage := func(page int) ([]parser.Ride, int, error) {
		f2 := f
		f2.Page = page
		return s.ListRides(f2)
	}

	rides, total, err := s.ListRides(f)
	if err != nil {
		return err
	}

	if display.IsTTY() && !jsonOutput {
		selected, err := display.RunRidesTUI(os.Stdout, rides, total, ridesLimit, loadPage)
		if err != nil {
			return err
		}
		if selected != nil {
			display.PrintRideDetail(os.Stdout, *selected, false, cfg.Units)
		}
		return nil
	}

	display.PrintRideList(os.Stdout, rides, total, ridesPage, ridesLimit, jsonOutput, cfg.Units)
	return nil
}
