package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Show personal records (all-time bests)",
	RunE:  runRecords,
}

var (
	recordsYear  int
	recordsMonth int
	recordsWeek  int
)

func init() {
	rootCmd.AddCommand(recordsCmd)
	recordsCmd.Flags().IntVar(&recordsYear, "year", 0, "filter by year (e.g. 2024)")
	recordsCmd.Flags().IntVar(&recordsMonth, "month", 0, "filter by month (1-12)")
	recordsCmd.Flags().IntVar(&recordsWeek, "week", 0, "filter by ISO week number (1-53)")
}

func runRecords(cmd *cobra.Command, args []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	f := store.RecordsFilters{}

	noFlags := recordsYear == 0 && recordsMonth == 0 && recordsWeek == 0

	if recordsYear != 0 {
		f.Year = &recordsYear
	}
	if recordsMonth != 0 {
		f.Month = &recordsMonth
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}
	if recordsWeek != 0 {
		f.Week = &recordsWeek
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}

	var label string
	if noFlags {
		label = "all time"
	} else {
		var labelParts []string
		if f.Year != nil {
			labelParts = append(labelParts, fmt.Sprintf("%d", *f.Year))
		}
		if f.Month != nil {
			labelParts = append(labelParts, fmt.Sprintf("month %02d", *f.Month))
		}
		if f.Week != nil {
			labelParts = append(labelParts, fmt.Sprintf("week %d", *f.Week))
		}
		label = strings.Join(labelParts, " ")
		if label == "" {
			label = "all time"
		}
	}

	recs, err := s.GetRecords(f)
	if err != nil {
		return err
	}

	display.PrintRecords(os.Stdout, recs, label, jsonOutput, cfg.Units)
	return nil
}
