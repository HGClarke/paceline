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

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregated ride totals",
	RunE:  runStats,
}

var (
	statsYear  int
	statsMonth int
	statsWeek  int
	statsFrom  string
	statsTo    string
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsYear, "year", 0, "filter by year (e.g. 2024)")
	statsCmd.Flags().IntVar(&statsMonth, "month", 0, "filter by month (1-12)")
	statsCmd.Flags().IntVar(&statsWeek, "week", 0, "filter by ISO week number (1-53)")
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
}

func runStats(cmd *cobra.Command, args []string) error {
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
	f := store.StatsFilters{}

	noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0 && statsFrom == "" && statsTo == ""

	if statsYear != 0 {
		f.Year = &statsYear
	}
	if statsMonth != 0 {
		f.Month = &statsMonth
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}
	if statsWeek != 0 {
		f.Week = &statsWeek
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
	}
	f.From, f.To, err = parseDateRange(statsFrom, statsTo)
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

	st, err := s.GetStats(f)
	if err != nil {
		return err
	}

	display.PrintStats(os.Stdout, st, label, jsonOutput, cfg.Units)
	return nil
}
