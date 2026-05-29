package cmd

import (
	"fmt"
	"time"
)

// parseDateRange parses --from and --to flag strings into time.Time pointers.
// Either or both may be empty (open-ended range). Returns an error if either
// string is not YYYY-MM-DD or if from is after to.
func parseDateRange(fromStr, toStr string) (from, to *time.Time, err error) {
	if fromStr != "" {
		t, parseErr := time.Parse("2006-01-02", fromStr)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid --from %q: use YYYY-MM-DD", fromStr)
		}
		from = &t
	}
	if toStr != "" {
		t, parseErr := time.Parse("2006-01-02", toStr)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid --to %q: use YYYY-MM-DD", toStr)
		}
		to = &t
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, nil, fmt.Errorf("--from must not be after --to")
	}
	return from, to, nil
}

// dateRangeLabel returns the label segment for active From/To filters.
// Returns empty string when neither is set.
func dateRangeLabel(from, to *time.Time) string {
	if from != nil && to != nil {
		return fmt.Sprintf("%s to %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	if from != nil {
		return fmt.Sprintf("from %s", from.Format("2006-01-02"))
	}
	if to != nil {
		return fmt.Sprintf("to %s", to.Format("2006-01-02"))
	}
	return ""
}
