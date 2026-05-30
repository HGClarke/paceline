# Year-over-Year Comparison Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `--compare <year>` flag to `paceline stats` that renders a 4-column side-by-side table (Metric | Year1 | Year2 | Δ) for rides, distance, duration, and elevation totals.

**Architecture:** `GetStats` is called twice (primary and compare filters) unchanged; a new `PrintStatsComparison` display function renders the comparison table with a `formatStatsDelta` helper; `cmd/stats.go` gains validation logic, a second `GetStats` call, and routing to the new display function.

**Tech Stack:** Go, `github.com/olekukonko/tablewriter`, DuckDB (`database/sql`), Cobra

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/display/table.go` | Modify | Add `formatStatsDelta` helper and `PrintStatsComparison` function |
| `internal/display/display_test.go` | Modify | Add tests for `formatStatsDelta` and `PrintStatsComparison` (table + JSON) |
| `cmd/stats.go` | Modify | Add `--compare` flag, validation, second `GetStats` call, routing |

---

## Task 1: Display layer — `formatStatsDelta` + `PrintStatsComparison`

**Files:**
- Modify: `internal/display/display_test.go`
- Modify: `internal/display/table.go`

- [ ] **Step 1: Write failing tests**

Add the following to `internal/display/display_test.go`. The file is `package display`, so unexported helpers are accessible.

```go
func TestFormatStatsDelta_Positive(t *testing.T) {
	got := formatStatsDelta(100000, 80000, func(v float64) string { return FormatDistance(v, "metric") })
	// delta = 20000 m = +20.0 km; pct = 20000/80000*100 = 25%
	if !strings.Contains(got, "+20.0 km") {
		t.Errorf("expected '+20.0 km' in %q", got)
	}
	if !strings.Contains(got, "+25%") {
		t.Errorf("expected '+25%%' in %q", got)
	}
}

func TestFormatStatsDelta_Negative(t *testing.T) {
	got := formatStatsDelta(80000, 100000, func(v float64) string { return FormatDistance(v, "metric") })
	// delta = -20000 m = -20.0 km; pct = -20000/100000*100 = -20%
	if !strings.Contains(got, "-20.0 km") {
		t.Errorf("expected '-20.0 km' in %q", got)
	}
	if !strings.Contains(got, "-20%") {
		t.Errorf("expected '-20%%' in %q", got)
	}
}

func TestFormatStatsDelta_ZeroBase(t *testing.T) {
	got := formatStatsDelta(5, 0, func(v float64) string { return strconv.Itoa(int(v)) })
	// base is zero: no percentage, just "+5"
	if got != "+5" {
		t.Errorf("formatStatsDelta zero base = %q, want %q", got, "+5")
	}
}

func TestPrintStatsComparison_Table(t *testing.T) {
	st1 := store.Stats{
		RideCount:       10,
		TotalDistanceM:  100000,
		TotalDurationS:  3600,  // "1h 00m 00s"
		TotalElevationM: 1000,
	}
	st2 := store.Stats{
		RideCount:       8,
		TotalDistanceM:  80000,
		TotalDurationS:  3200, // "53m 20s"
		TotalElevationM: 800,
	}
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", false, "metric")
	out := buf.String()

	for _, want := range []string{
		"2025 vs 2024",
		"1h 00m 00s",  // st1 duration
		"53m 20s",     // st2 duration
		"100.0 km",    // st1 distance
		"80.0 km",     // st2 distance
		"+20.0 km",    // distance delta value
		"+6m 40s",     // duration delta (delta = 400s = 6m40s)
		"+200 m",      // elevation delta value
		"+25%",        // rides / distance / elevation pct
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestPrintStatsComparison_ZeroBase(t *testing.T) {
	st1 := store.Stats{
		RideCount:       5,
		TotalDistanceM:  50000,
		TotalDurationS:  1800, // "30m 00s"
		TotalElevationM: 500,
	}
	st2 := store.Stats{} // all zeros
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", false, "metric")
	out := buf.String()

	for _, want := range []string{"+5", "+50.0 km", "+500 m"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
	if strings.Contains(out, "%") {
		t.Errorf("expected no percentage when base is zero, got:\n%s", out)
	}
}

func TestPrintStatsComparison_JSON(t *testing.T) {
	st1 := store.Stats{RideCount: 10, TotalDistanceM: 100000}
	st2 := store.Stats{RideCount: 8, TotalDistanceM: 80000}
	var buf bytes.Buffer
	PrintStatsComparison(&buf, st1, st2, "2025", "2024", true, "metric")

	var out struct {
		Primary struct {
			Label string      `json:"label"`
			Stats store.Stats `json:"stats"`
		} `json:"primary"`
		Compare struct {
			Label string      `json:"label"`
			Stats store.Stats `json:"stats"`
		} `json:"compare"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if out.Primary.Label != "2025" {
		t.Errorf("primary label = %q, want %q", out.Primary.Label, "2025")
	}
	if out.Compare.Label != "2024" {
		t.Errorf("compare label = %q, want %q", out.Compare.Label, "2024")
	}
	if out.Primary.Stats.RideCount != 10 {
		t.Errorf("primary ride count = %d, want 10", out.Primary.Stats.RideCount)
	}
	if out.Compare.Stats.RideCount != 8 {
		t.Errorf("compare ride count = %d, want 8", out.Compare.Stats.RideCount)
	}
}
```

`TestFormatStatsDelta_ZeroBase` uses `strconv.Itoa`, so add `"strconv"` and `"encoding/json"` to the import block in `display_test.go` if not already present.

- [ ] **Step 2: Run failing tests**

```bash
go test ./internal/display/... -run "TestFormatStatsDelta|TestPrintStatsComparison" -v
```

Expected: `FAIL — undefined: formatStatsDelta`, `undefined: PrintStatsComparison`

- [ ] **Step 3: Add `formatStatsDelta` to `internal/display/table.go`**

Add after the `FormatElevation` function at the bottom of `internal/display/table.go`:

```go
// formatStatsDelta formats the signed delta between v1 and v2 as "+X unit (+pct%)" or "-X unit (-pct%)".
// formatAbs formats the absolute (non-negative) delta value with its unit.
// When v2 is zero, the percentage is omitted.
func formatStatsDelta(v1, v2 float64, formatAbs func(float64) string) string {
	delta := v1 - v2
	abs := math.Abs(delta)
	var sign string
	switch {
	case delta > 0:
		sign = "+"
	case delta < 0:
		sign = "-"
	}
	formatted := sign + formatAbs(abs)
	if v2 == 0 {
		return formatted
	}
	pct := delta / v2 * 100
	absPct := math.Abs(pct)
	if pct >= 0 {
		return fmt.Sprintf("%s (+%.0f%%)", formatted, absPct)
	}
	return fmt.Sprintf("%s (-%.0f%%)", formatted, absPct)
}
```

(`math` and `fmt` are already imported in `table.go`.)

- [ ] **Step 4: Add `PrintStatsComparison` to `internal/display/table.go`**

Add after `PrintStats`:

```go
// PrintStatsComparison renders a side-by-side comparison of two Stats periods.
// label1 and label2 are the display names for the primary and compare periods (e.g. "2025", "2024").
// Only total metrics are shown: rides, distance, duration, elevation.
func PrintStatsComparison(w io.Writer, st1, st2 store.Stats, label1, label2 string, jsonOut bool, units string) {
	if jsonOut {
		type entry struct {
			Label string      `json:"label"`
			Stats store.Stats `json:"stats"`
		}
		out := struct {
			Primary entry `json:"primary"`
			Compare entry `json:"compare"`
		}{
			Primary: entry{Label: label1, Stats: st1},
			Compare: entry{Label: label2, Stats: st2},
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}

	fmt.Fprintf(w, "Stats: %s vs %s\n\n", label1, label2)

	ridesDelta := formatStatsDelta(
		float64(st1.RideCount), float64(st2.RideCount),
		func(v float64) string { return strconv.Itoa(int(v)) },
	)
	distDelta := formatStatsDelta(
		st1.TotalDistanceM, st2.TotalDistanceM,
		func(v float64) string { return FormatDistance(v, units) },
	)
	durDelta := formatStatsDelta(
		float64(st1.TotalDurationS), float64(st2.TotalDurationS),
		func(v float64) string { return formatDuration(int(v)) },
	)
	elevDelta := formatStatsDelta(
		st1.TotalElevationM, st2.TotalElevationM,
		func(v float64) string { return FormatElevation(v, units) },
	)

	table := tablewriter.NewWriter(w)
	table.Options(
		tablewriter.WithBorders(tw.Border{ //nolint:staticcheck // SA1019: WithBorders deprecated but replacement API not yet stable
			Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
		}),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header([]string{"Metric", label1, label2, "Δ"})
	_ = table.Bulk([][]string{
		{"Rides", strconv.Itoa(st1.RideCount), strconv.Itoa(st2.RideCount), ridesDelta},
		{"Total Distance", FormatDistance(st1.TotalDistanceM, units), FormatDistance(st2.TotalDistanceM, units), distDelta},
		{"Total Duration", formatDuration(st1.TotalDurationS), formatDuration(st2.TotalDurationS), durDelta},
		{"Total Elevation", FormatElevation(st1.TotalElevationM, units), FormatElevation(st2.TotalElevationM, units), elevDelta},
	})
	_ = table.Render()
}
```

(`json`, `strconv`, `tablewriter`, `tw` are all already imported in `table.go`.)

- [ ] **Step 5: Run tests to confirm they pass**

```bash
go test ./internal/display/... -run "TestFormatStatsDelta|TestPrintStatsComparison" -v
```

Expected: all 5 tests PASS.

- [ ] **Step 6: Run full suite**

```bash
make all
```

Expected: all tests pass, no vet or lint errors.

- [ ] **Step 7: Commit**

```bash
git add internal/display/table.go internal/display/display_test.go
git commit -m "feat(display): add PrintStatsComparison for year-over-year stats output"
```

---

## Task 2: `--compare` flag in `cmd/stats.go`

**Files:**
- Modify: `cmd/stats.go`

- [ ] **Step 1: Add `statsCompare` variable and `--compare` flag**

In `cmd/stats.go`, add `statsCompare int` to the existing `var` block:

```go
var (
	statsYear    int
	statsMonth   int
	statsWeek    int
	statsFrom    string
	statsTo      string
	statsCompare int
)
```

Add the flag registration to `init()`:

```go
statsCmd.Flags().IntVar(&statsCompare, "compare", 0, "compare to this year (e.g. --year 2025 --compare 2024)")
```

- [ ] **Step 2: Add `strconv` import**

In the import block of `cmd/stats.go`, add `"strconv"`:

```go
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
```

- [ ] **Step 3: Add compare path to `runStats`**

In `runStats`, after the `parseDateRange` call and before the label-building block, insert:

```go
if statsCompare != 0 {
	// Default primary year to current if not explicitly set.
	if f.Year == nil {
		y := now.Year()
		f.Year = &y
	}
	if statsWeek != 0 || statsFrom != "" || statsTo != "" {
		return fmt.Errorf("--compare is only supported with --year and --month")
	}
	if statsCompare == *f.Year {
		return fmt.Errorf("--compare year must differ from the primary year")
	}

	cmpF := store.StatsFilters{Year: &statsCompare}
	if f.Month != nil {
		cmpF.Month = f.Month
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
```

The existing label-building block and `GetStats` call below are unchanged — they handle the non-compare path.

- [ ] **Step 4: Run full suite**

```bash
make all
```

Expected: all tests pass, no vet or lint errors.

- [ ] **Step 5: Smoke test**

With a populated database, verify each case:

```bash
# Year comparison (--year defaults to current year)
go run . stats --compare 2024

# Explicit primary year
go run . stats --year 2025 --compare 2024

# Month comparison (same month, two years)
go run . stats --month 5 --year 2025 --compare 2024

# JSON output
go run . stats --year 2025 --compare 2024 --json

# Error: same year
go run . stats --year 2025 --compare 2025
# Expected: "Error: --compare year must differ from the primary year"

# Error: incompatible flag
go run . stats --compare 2024 --week 22
# Expected: "Error: --compare is only supported with --year and --month"
```

- [ ] **Step 6: Commit**

```bash
git add cmd/stats.go
git commit -m "feat(cmd): add --compare flag to stats for year-over-year comparison"
```

---

## Task 3: Update feature roadmap and README

**Files:**
- Modify: `docs/feature-roadmap.md`
- Modify: `README.md`

- [ ] **Step 1: Mark feature #11 complete in the roadmap**

In `docs/feature-roadmap.md`, update the priority matrix row for feature 11:

```markdown
| 11 | [Year-over-year comparison](#11-year-over-year-comparison) | 🟠 Medium | Medium | ✅ Completed |
```

- [ ] **Step 2: Add a status note under the feature description**

In the `### 11. Year-over-Year Comparison` section, add after the proposed commands block:

```markdown
> **Status: Completed.** `--compare <year>` is live on `stats`. Compares totals (rides, distance, duration, elevation) side-by-side with a Δ column. Supports `--year`/`--month` filters; `--week` and date ranges are not supported with `--compare`.
```

- [ ] **Step 3: Update README**

In the `README.md` stats section, add the comparison usage. Find the existing stats examples and add:

```markdown
# Year-over-year comparison
paceline stats --year 2025 --compare 2024

# Same month, different years
paceline stats --month 5 --year 2025 --compare 2024
```

- [ ] **Step 4: Commit**

```bash
git add docs/feature-roadmap.md README.md
git commit -m "docs: mark year-over-year comparison complete; update README"
```
