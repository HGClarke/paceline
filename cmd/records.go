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
	if recordsFrom != "" {
		t, err := time.Parse("2006-01-02", recordsFrom)
		if err != nil {
			return fmt.Errorf("invalid --from %q: use YYYY-MM-DD", recordsFrom)
		}
		f.From = &t
	}
	if recordsTo != "" {
		t, err := time.Parse("2006-01-02", recordsTo)
		if err != nil {
			return fmt.Errorf("invalid --to %q: use YYYY-MM-DD", recordsTo)
		}
		f.To = &t
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return fmt.Errorf("--from must not be after --to")
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
		if f.From != nil && f.To != nil { //nolint:gocritic // ifElseChain: switch not applicable for pointer comparisons
			labelParts = append(labelParts, fmt.Sprintf("%s to %s", f.From.Format("2006-01-02"), f.To.Format("2006-01-02")))
		} else if f.From != nil {
			labelParts = append(labelParts, fmt.Sprintf("from %s", f.From.Format("2006-01-02")))
		} else if f.To != nil {
			labelParts = append(labelParts, fmt.Sprintf("to %s", f.To.Format("2006-01-02")))
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
