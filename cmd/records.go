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
	recordsFrom  string
	recordsTo    string
)

func init() {
	rootCmd.AddCommand(recordsCmd)
	recordsCmd.Flags().IntVar(&recordsYear, "year", 0, "filter by year (e.g. 2024)")
	recordsCmd.Flags().IntVar(&recordsMonth, "month", 0, "filter by month (1-12)")
	recordsCmd.Flags().IntVar(&recordsWeek, "week", 0, "filter by ISO week number (1-53)")
	recordsCmd.Flags().StringVar(&recordsFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	recordsCmd.Flags().StringVar(&recordsTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
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

	noFlags := recordsYear == 0 && recordsMonth == 0 && recordsWeek == 0 && recordsFrom == "" && recordsTo == ""

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
	f.From, f.To, err = parseDateRange(recordsFrom, recordsTo)
	if err != nil {
		return err
	}

	// Build human-readable label from the active filters.
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
		if part := dateRangeLabel(f.From, f.To); part != "" {
			labelParts = append(labelParts, part)
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
