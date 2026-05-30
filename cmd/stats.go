package cmd

import (
	"fmt"
	"os"
	"strconv"
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
	statsYear    int
	statsMonth   int
	statsWeek    int
	statsFrom    string
	statsTo      string
	statsCompare int
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsYear, "year", 0, "filter by year (e.g. 2024)")
	statsCmd.Flags().IntVar(&statsMonth, "month", 0, "filter by month (1-12)")
	statsCmd.Flags().IntVar(&statsWeek, "week", 0, "filter by ISO week number (1-53)")
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "filter rides on or after this date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "filter rides on or before this date (YYYY-MM-DD)")
	statsCmd.Flags().IntVar(&statsCompare, "compare", 0, "compare to this year (e.g. --year 2025 --compare 2024)")
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

	if statsCompare != 0 {
		// Default primary year to current if not explicitly set.
		if f.Year == nil {
			y := now.Year()
			f.Year = &y
		}
		if err := validateCompareArgs(*f.Year, statsCompare, statsWeek, statsFrom, statsTo); err != nil {
			return err
		}

		cmpF := store.StatsFilters{Year: &statsCompare}
		if f.Month != nil {
			m := *f.Month
			cmpF.Month = &m
		}

		st1, err := s.GetStats(f)
		if err != nil {
			return err
		}
		st2, err := s.GetStats(cmpF)
		if err != nil {
			return err
		}

		primaryYear := *f.Year
		var label1, label2 string
		if f.Month != nil {
			monthName := time.Month(*f.Month).String()
			label1 = fmt.Sprintf("%s %d", monthName, primaryYear)
			label2 = fmt.Sprintf("%s %d", monthName, statsCompare)
		} else {
			label1 = strconv.Itoa(primaryYear)
			label2 = strconv.Itoa(statsCompare)
		}

		display.PrintStatsComparison(os.Stdout, st1, st2, label1, label2, jsonOutput, cfg.Units)
		return nil
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

// validateCompareArgs returns an error if the --compare flag is combined with
// incompatible flags or if both years are the same.
func validateCompareArgs(primaryYear, compareYear, statsWeek int, statsFrom, statsTo string) error {
	if statsWeek != 0 || statsFrom != "" || statsTo != "" {
		return fmt.Errorf("--compare is only supported with --year and --month")
	}
	if compareYear == primaryYear {
		return fmt.Errorf("--compare year must differ from the primary year")
	}
	return nil
}
