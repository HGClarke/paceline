package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hollandclarke/paceline/internal/display"
	"github.com/hollandclarke/paceline/internal/store"
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
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsYear, "year", 0, "filter by year (e.g. 2024)")
	statsCmd.Flags().IntVar(&statsMonth, "month", 0, "filter by month (1-12)")
	statsCmd.Flags().IntVar(&statsWeek, "week", 0, "filter by ISO week number (1-53)")
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
	label := "current month"

	noFlags := statsYear == 0 && statsMonth == 0 && statsWeek == 0
	if noFlags {
		y, m := now.Year(), int(now.Month())
		f.Year = &y
		f.Month = &m
	} else {
		if statsYear != 0 {
			f.Year = &statsYear
			label = fmt.Sprintf("year %d", statsYear)
		}
		if statsMonth != 0 {
			f.Month = &statsMonth
			if f.Year == nil {
				y := now.Year()
				f.Year = &y
			}
			label = fmt.Sprintf("%s %02d", label, statsMonth)
		}
		if statsWeek != 0 {
			f.Week = &statsWeek
			if f.Year == nil {
				y := now.Year()
				f.Year = &y
			}
			label = fmt.Sprintf("%s week %d", label, statsWeek)
		}
	}

	st, err := s.GetStats(f)
	if err != nil {
		return err
	}

	display.PrintStats(os.Stdout, st, label, jsonOutput)
	return nil
}
