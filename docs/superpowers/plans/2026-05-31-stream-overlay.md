# Stream Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--overlay` flag to `ride <id> stream` that renders multiple fields on one overlaid ASCII chart; color all charts and simplify captions.

**Architecture:** Update `display.PrintStreamChart` to accept `[][]parser.Stream` + `[]string` fields, rendering via `asciigraph.Plot` (single) or `asciigraph.PlotMany` (overlay). Update `cmd/stream.go` to add the `--overlay` bool flag and construct the new call signature in both code paths.

**Tech Stack:** `github.com/guptarohit/asciigraph` (`Plot`, `PlotMany`, `SeriesColors`, `SeriesLegends`, `Caption`), Cobra, Go standard library.

---

### Task 1: Update `PrintStreamChart` — new signature and single-field rendering with color

**Files:**
- Modify: `internal/display/chart.go`
- Modify: `internal/display/display_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/display/display_test.go` (after the existing `TestExtractFieldValues` block):

```go
func TestPrintStreamChart_SingleField(t *testing.T) {
	power := 250
	points := []parser.Stream{
		{PowerW: &power},
		{PowerW: &power},
		{PowerW: &power},
	}
	var buf bytes.Buffer
	PrintStreamChart(&buf, [][]parser.Stream{points}, []string{"power"})
	out := buf.String()
	if !strings.Contains(out, "power") {
		t.Errorf("expected caption 'power' in output, got:\n%s", out)
	}
	if strings.Contains(out, "points") {
		t.Errorf("expected no point count in caption, got:\n%s", out)
	}
}

func TestPrintStreamChart_SingleField_NoData(t *testing.T) {
	var buf bytes.Buffer
	PrintStreamChart(&buf, [][]parser.Stream{{}}, []string{"power"})
	out := buf.String()
	if !strings.Contains(out, "No power data") {
		t.Errorf("expected no-data message, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/display/... -run TestPrintStreamChart -v
```

Expected: compile error — `PrintStreamChart` has wrong argument types.

- [ ] **Step 3: Rewrite `internal/display/chart.go`**

Replace the entire file with:

```go
package display

import (
	"fmt"
	"io"

	"github.com/guptarohit/asciigraph"
	"github.com/HGClarke/paceline/internal/parser"
)

var fieldColor = map[string]asciigraph.AnsiColor{
	"power":    asciigraph.Red,
	"hr":       asciigraph.Blue,
	"speed":    asciigraph.Green,
	"cadence":  asciigraph.Yellow,
	"altitude": asciigraph.Cyan,
}

// PrintStreamChart renders a terminal line chart for one or more stream fields.
// allSeries[i] contains the stream points for fields[i].
// Single field: uses asciigraph.Plot with color. Multiple fields: uses PlotMany with legends.
func PrintStreamChart(w io.Writer, allSeries [][]parser.Stream, fields []string) {
	if len(fields) == 1 {
		data := extractFieldValues(allSeries[0], fields[0])
		if len(data) == 0 {
			fmt.Fprintf(w, "No %s data to display.\n", fields[0])
			return
		}
		chart := asciigraph.Plot(data,
			asciigraph.Height(15),
			asciigraph.Width(80),
			asciigraph.Caption(fields[0]),
			asciigraph.SeriesColors(fieldColor[fields[0]]),
		)
		fmt.Fprintln(w, chart)
		return
	}

	var seriesData [][]float64
	var colors []asciigraph.AnsiColor
	for i, field := range fields {
		seriesData = append(seriesData, extractFieldValues(allSeries[i], field))
		colors = append(colors, fieldColor[field])
	}
	chart := asciigraph.PlotMany(seriesData,
		asciigraph.Height(15),
		asciigraph.Width(80),
		asciigraph.Caption("stream overlay"),
		asciigraph.SeriesLegends(fields...),
		asciigraph.SeriesColors(colors...),
	)
	fmt.Fprintln(w, chart)
}

func extractFieldValues(points []parser.Stream, field string) []float64 {
	var vals []float64
	for _, p := range points {
		switch field {
		case "power":
			if p.PowerW != nil {
				vals = append(vals, float64(*p.PowerW))
			}
		case "hr":
			if p.HRBPM != nil {
				vals = append(vals, float64(*p.HRBPM))
			}
		case "speed":
			if p.SpeedMPS != nil {
				vals = append(vals, *p.SpeedMPS*3.6)
			}
		case "cadence":
			if p.CadenceRPM != nil {
				vals = append(vals, float64(*p.CadenceRPM))
			}
		case "altitude":
			if p.AltitudeM != nil {
				vals = append(vals, *p.AltitudeM)
			}
		}
	}
	return vals
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/display/... -run TestPrintStreamChart -v
```

Expected: PASS for both `TestPrintStreamChart_SingleField` and `TestPrintStreamChart_SingleField_NoData`.

- [ ] **Step 5: Run the full display test suite to check for regressions**

```bash
go test ./internal/display/... -v
```

Expected: all tests PASS. Note: `TestExtractFieldValues` calls `extractFieldValues` directly — that function is unchanged, so it should still pass.

- [ ] **Step 6: Commit**

```bash
git add internal/display/chart.go internal/display/display_test.go
git commit -m "feat(display): update PrintStreamChart signature; add color and simplified captions"
```

---

### Task 2: Add overlay rendering test

**Files:**
- Modify: `internal/display/display_test.go`

(The overlay code path was already implemented in Task 1's `chart.go`. This task adds the test coverage.)

- [ ] **Step 1: Write the failing overlay test**

Add to `internal/display/display_test.go` after the `TestPrintStreamChart_SingleField_NoData` block:

```go
func TestPrintStreamChart_Overlay(t *testing.T) {
	power := 250
	hr := 160
	powerPoints := []parser.Stream{
		{PowerW: &power},
		{PowerW: &power},
		{PowerW: &power},
	}
	hrPoints := []parser.Stream{
		{HRBPM: &hr},
		{HRBPM: &hr},
		{HRBPM: &hr},
	}
	var buf bytes.Buffer
	PrintStreamChart(&buf, [][]parser.Stream{powerPoints, hrPoints}, []string{"power", "hr"})
	out := buf.String()
	if !strings.Contains(out, "stream overlay") {
		t.Errorf("expected caption 'stream overlay', got:\n%s", out)
	}
	if !strings.Contains(out, "power") {
		t.Errorf("expected 'power' in legend, got:\n%s", out)
	}
	if !strings.Contains(out, "hr") {
		t.Errorf("expected 'hr' in legend, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run the test**

```bash
go test ./internal/display/... -run TestPrintStreamChart_Overlay -v
```

Expected: PASS (implementation already exists from Task 1).

- [ ] **Step 3: Commit**

```bash
git add internal/display/display_test.go
git commit -m "test(display): add overlay chart test"
```

---

### Task 3: Update `cmd/stream.go` with `--overlay` flag

**Files:**
- Modify: `cmd/stream.go`

- [ ] **Step 1: Add `--overlay` flag and rewrite `runStream`**

Replace the entire contents of `cmd/stream.go` with:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/parser"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var streamFields []string
var streamOverlay bool

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Show a time-series chart for a ride stream",
	Args:  cobra.NoArgs,
	RunE:  runStream,
}

func init() {
	streamCmd.Flags().StringSliceVar(&streamFields, "field", nil, "field(s) to chart: power, hr, speed, cadence, altitude (repeatable or comma-separated)")
	streamCmd.Flags().BoolVar(&streamOverlay, "overlay", false, "render all fields on a single overlaid chart")
}

func runStream(cmd *cobra.Command, args []string) error {
	id := currentRideID

	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	fields := streamFields
	if len(fields) == 0 {
		available, err := s.AvailableFields(id)
		if err != nil {
			return err
		}
		for _, candidate := range []string{"power", "hr", "speed"} {
			for _, a := range available {
				if a == candidate {
					fields = []string{candidate}
					break
				}
			}
			if len(fields) > 0 {
				break
			}
		}
		if len(fields) == 0 {
			return fmt.Errorf("ride #%d has no stream data", currentRide.Position)
		}
	}

	if streamOverlay {
		var allSeries [][]parser.Stream
		var validFields []string
		for _, field := range fields {
			points, err := s.GetStreams(id, field)
			if err != nil {
				return err
			}
			if len(points) == 0 {
				available, _ := s.AvailableFields(id)
				fmt.Fprintf(os.Stderr, "No %s data for ride #%d. Available fields: %v\n", field, currentRide.Position, available)
				continue
			}
			allSeries = append(allSeries, points)
			validFields = append(validFields, field)
		}
		if len(validFields) == 0 {
			return fmt.Errorf("ride #%d has no stream data for requested fields", currentRide.Position)
		}
		display.PrintStreamChart(os.Stdout, allSeries, validFields)
		return nil
	}

	for i, field := range fields {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		points, err := s.GetStreams(id, field)
		if err != nil {
			return err
		}
		if len(points) == 0 {
			available, _ := s.AvailableFields(id)
			fmt.Fprintf(os.Stderr, "No %s data for ride #%d. Available fields: %v\n", field, currentRide.Position, available)
			continue
		}
		display.PrintStreamChart(os.Stdout, [][]parser.Stream{points}, []string{field})
	}
	return nil
}
```

- [ ] **Step 2: Build to confirm compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all tests**

```bash
make all
```

Expected: vet, tests, and lint all pass.

- [ ] **Step 4: Smoke test single-field (no flag)**

```bash
go run . ride 1 stream --field=power
```

Expected: colored single-field chart with caption `"power"` and no point count.

- [ ] **Step 5: Smoke test overlay**

```bash
go run . ride 1 stream --field=power --field=hr --overlay
```

Expected: single overlaid chart with caption `"stream overlay"` and a colored legend showing `■ power   ■ hr`.

- [ ] **Step 6: Smoke test multi-field without overlay (separate charts)**

```bash
go run . ride 1 stream --field=power --field=hr
```

Expected: two separate colored charts printed sequentially, each with just the field name as caption.

- [ ] **Step 7: Commit**

```bash
git add cmd/stream.go
git commit -m "feat(cmd): add --overlay flag to stream command for multi-field overlaid charts"
```
