# Route Map in Terminal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `paceline ride <id> route` subcommand that renders the ride's GPS track as a Braille Unicode art map with elevation-based colour gradient.

**Architecture:** The store layer exposes a `GetGPSPoints` query; a self-contained Braille canvas (no external library, ~60 lines) handles sub-pixel rendering and per-cell ANSI colouring; the display layer projects lat/lon onto the canvas with Mercator correction and maps altitude to a 5-stop colour gradient; the command layer wires everything together under `ride <id> route`.

**Tech Stack:** Go stdlib, DuckDB (existing), ANSI 256-colour escape codes, Unicode Braille block U+2800–U+28FF, existing `tablewriter` and `asciigraph` patterns for display conventions.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/store/streams.go` | Modify | Add `GPSPoint` type and `GetGPSPoints` method |
| `internal/store/streams_test.go` | Modify | Tests for `GetGPSPoints` |
| `internal/display/route.go` | Create | Braille canvas + `PrintRoute` function |
| `internal/display/display_test.go` | Modify | Tests for `PrintRoute` |
| `cmd/route.go` | Create | `routeCmd` definition and `runRoute` handler |
| `cmd/ride.go` | Modify | `rideCmd.AddCommand(routeCmd)` in `init()` |
| `README.md` | Modify | Add `route` command docs |
| `docs/feature-roadmap.md` | Modify | Mark #15 completed |

---

## Task 1: Store — GPSPoint type and GetGPSPoints

**Files:**
- Modify: `internal/store/streams.go`
- Modify: `internal/store/streams_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/store/streams_test.go`:

```go
func TestGetGPSPoints_NoData(t *testing.T) {
	s := openTestStore(t)
	id := insertTestRide(t, s)
	// Insert a stream point with no lat/lon
	_, err := s.db.Exec(`INSERT INTO streams (ride_id, timestamp, elapsed_s, speed_mps)
		VALUES (?, NOW(), 0, 5.0)`, id)
	require.NoError(t, err)

	pts, err := s.GetGPSPoints(id)
	require.NoError(t, err)
	assert.Empty(t, pts)
}

func TestGetGPSPoints_WithData(t *testing.T) {
	s := openTestStore(t)
	id := insertTestRide(t, s)
	alt := 120.5
	_, err := s.db.Exec(`INSERT INTO streams (ride_id, timestamp, elapsed_s, lat, lon, altitude_m)
		VALUES (?, NOW(), 0, 51.5074, -0.1278, ?)`, id, alt)
	require.NoError(t, err)
	_, err = s.db.Exec(`INSERT INTO streams (ride_id, timestamp, elapsed_s, lat, lon, altitude_m)
		VALUES (?, NOW(), 1, 51.5080, -0.1270, NULL)`, id)
	require.NoError(t, err)

	pts, err := s.GetGPSPoints(id)
	require.NoError(t, err)
	require.Len(t, pts, 2)

	assert.Equal(t, 0, pts[0].ElapsedS)
	assert.InDelta(t, 51.5074, pts[0].Lat, 0.0001)
	assert.InDelta(t, -0.1278, pts[0].Lon, 0.0001)
	assert.NotNil(t, pts[0].AltitudeM)
	assert.InDelta(t, 120.5, *pts[0].AltitudeM, 0.01)

	assert.Equal(t, 1, pts[1].ElapsedS)
	assert.Nil(t, pts[1].AltitudeM)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/store/... -run TestGetGPSPoints -v
```

Expected: `FAIL — GetGPSPoints undefined`

- [ ] **Step 3: Add GPSPoint type and GetGPSPoints to internal/store/streams.go**

Add after the existing `fieldColumn` function:

```go
// GPSPoint is a single GPS sample from the streams table.
type GPSPoint struct {
	ElapsedS  int
	Lat       float64
	Lon       float64
	AltitudeM *float64
}

// GetGPSPoints returns all GPS-tagged stream points for a ride, ordered by elapsed time.
// Points without lat/lon are excluded. Returns an empty slice (not an error) when no GPS data exists.
func (s *Store) GetGPSPoints(rideID int64) ([]GPSPoint, error) {
	rows, err := s.db.Query(`
		SELECT elapsed_s, lat, lon, altitude_m
		FROM streams
		WHERE ride_id = ? AND lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY elapsed_s ASC`, rideID)
	if err != nil {
		return nil, fmt.Errorf("get gps points: %w", err)
	}
	defer rows.Close()

	var pts []GPSPoint
	for rows.Next() {
		var p GPSPoint
		var alt *float64
		if err := rows.Scan(&p.ElapsedS, &p.Lat, &p.Lon, &alt); err != nil {
			return nil, fmt.Errorf("scan gps point: %w", err)
		}
		p.AltitudeM = alt
		pts = append(pts, p)
	}
	return pts, rows.Err()
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/store/... -run TestGetGPSPoints -v
```

Expected: `PASS`

- [ ] **Step 5: Run full suite to confirm no regressions**

```bash
make all
```

Expected: all green

- [ ] **Step 6: Commit**

```bash
git add internal/store/streams.go internal/store/streams_test.go
git commit -m "feat(store): add GPSPoint type and GetGPSPoints query"
```

---

## Task 2: Display — Braille canvas and PrintRoute

**Files:**
- Create: `internal/display/route.go`
- Modify: `internal/display/display_test.go`

### Background: how Braille encoding works

Each Unicode Braille character (U+2800+bitmask) encodes a 2-wide × 4-tall dot grid. Mapping:

```
  col 0  col 1
  bit 0  bit 3   row 0
  bit 1  bit 4   row 1
  bit 2  bit 5   row 2
  bit 6  bit 7   row 3
```

An 80-column × 30-row terminal becomes a 160×120 effective pixel canvas. Each terminal cell stores which of its 8 sub-pixels are lit, plus one ANSI colour for the whole cell.

### Elevation colour gradient (ANSI 256-colour)

Map normalised altitude (0.0–1.0) to 5 bands:

| Range | Colour | ANSI 256 code |
|-------|--------|--------------|
| 0.0–0.2 | dark green | `\033[38;5;22m` |
| 0.2–0.4 | bright green | `\033[38;5;46m` |
| 0.4–0.6 | yellow | `\033[38;5;226m` |
| 0.6–0.8 | orange | `\033[38;5;208m` |
| 0.8–1.0 | red | `\033[38;5;196m` |

When all `AltitudeM` values are nil, fall back to white (`\033[37m`).

- [ ] **Step 1: Write failing tests**

Add to `internal/display/display_test.go`:

```go
func TestPrintRoute_NoGPS(t *testing.T) {
	var buf bytes.Buffer
	display.PrintRoute(&buf, nil, 78, 28, "test_ride.fit")
	out := buf.String()
	assert.Contains(t, out, "No GPS data")
}

func TestPrintRoute_WithData(t *testing.T) {
	// A simple 5-point square loop
	alt0, alt1 := 100.0, 200.0
	pts := []store.GPSPoint{
		{ElapsedS: 0,  Lat: 51.50, Lon: -0.13, AltitudeM: &alt0},
		{ElapsedS: 60, Lat: 51.51, Lon: -0.13, AltitudeM: &alt1},
		{ElapsedS: 120, Lat: 51.51, Lon: -0.12, AltitudeM: &alt1},
		{ElapsedS: 180, Lat: 51.50, Lon: -0.12, AltitudeM: &alt0},
		{ElapsedS: 240, Lat: 51.50, Lon: -0.13, AltitudeM: &alt0},
	}
	var buf bytes.Buffer
	display.PrintRoute(&buf, pts, 78, 28, "test_ride.fit")
	out := buf.String()
	assert.Contains(t, out, "┌")
	assert.Contains(t, out, "┘")
	assert.Contains(t, out, "test_ride.fit")
	// Should contain at least one Braille character (U+2800 range)
	assert.True(t, strings.ContainsRune(out, '⠀'+1) || strings.ContainsAny(out, "⣿⠿⡇⢸"))
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/display/... -run TestPrintRoute -v
```

Expected: `FAIL — PrintRoute undefined`

- [ ] **Step 3: Create internal/display/route.go**

```go
package display

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/HGClarke/paceline/internal/store"
)

const resetColor = "\033[0m"

// PrintRoute renders a GPS route as a Braille Unicode art map to w.
// w and h are the terminal character dimensions of the data area.
// filename is shown in the footer.
func PrintRoute(w io.Writer, pts []store.GPSPoint, width, height int, filename string) {
	if len(pts) == 0 {
		fmt.Fprintln(w, "No GPS data available for this ride.")
		return
	}

	pixW := width * 2
	pixH := height * 4

	// Bounding box with 6% padding
	minLat, maxLat := pts[0].Lat, pts[0].Lat
	minLon, maxLon := pts[0].Lon, pts[0].Lon
	minAlt, maxAlt := math.MaxFloat64, -math.MaxFloat64
	hasAlt := false
	for _, p := range pts {
		if p.Lat < minLat { minLat = p.Lat }
		if p.Lat > maxLat { maxLat = p.Lat }
		if p.Lon < minLon { minLon = p.Lon }
		if p.Lon > maxLon { maxLon = p.Lon }
		if p.AltitudeM != nil {
			hasAlt = true
			if *p.AltitudeM < minAlt { minAlt = *p.AltitudeM }
			if *p.AltitudeM > maxAlt { maxAlt = *p.AltitudeM }
		}
	}
	latRange := (maxLat - minLat) * 1.06
	lonRange := (maxLon - minLon) * 1.06
	midLat := (minLat + maxLat) / 2
	midLon := (minLon + maxLon) / 2

	// Mercator longitude correction: 1° lon ≠ 1° lat in metres
	cosLat := math.Cos(midLat * math.Pi / 180)

	// Scale to fit pixel canvas preserving aspect ratio
	// Braille sub-pixels are 2-wide × 4-tall per char cell.
	// Terminal chars are ~2:1 tall:wide, so sub-pixels are physically square — no extra correction.
	scaleX := float64(pixW-1) / (lonRange * cosLat)
	scaleY := float64(pixH-1) / latRange
	scale := math.Min(scaleX, scaleY)

	project := func(p store.GPSPoint) (x, y int) {
		x = int(math.Round((p.Lon-midLon)*cosLat*scale)) + pixW/2
		y = int(math.Round(-(p.Lat-midLat)*scale)) + pixH/2
		return
	}

	altNorm := func(p store.GPSPoint) float64 {
		if !hasAlt || p.AltitudeM == nil || maxAlt == minAlt {
			return 0.5
		}
		return (*p.AltitudeM - minAlt) / (maxAlt - minAlt)
	}

	c := newBrailleCanvas(width, height)
	for i := 1; i < len(pts); i++ {
		x0, y0 := project(pts[i-1])
		x1, y1 := project(pts[i])
		norm := (altNorm(pts[i-1]) + altNorm(pts[i])) / 2
		color := altColor(norm, hasAlt)
		c.line(x0, y0, x1, y1, color)
	}

	fmt.Fprintln(w, "  ┌"+strings.Repeat("─", width)+"┐")
	for _, row := range c.frame() {
		fmt.Fprintf(w, "  │%s│\n", row)
	}
	fmt.Fprintln(w, "  └"+strings.Repeat("─", width)+"┘")
	legend := "  " + filename
	if hasAlt {
		legend += "   gradient: \033[38;5;22m▬\033[0m low → \033[38;5;226m▬\033[0m mid → \033[38;5;196m▬\033[0m high"
	}
	fmt.Fprintln(w, legend)
}

func altColor(norm float64, hasAlt bool) string {
	if !hasAlt {
		return "\033[37m" // white — no altitude data
	}
	switch {
	case norm < 0.2:
		return "\033[38;5;22m"
	case norm < 0.4:
		return "\033[38;5;46m"
	case norm < 0.6:
		return "\033[38;5;226m"
	case norm < 0.8:
		return "\033[38;5;208m"
	default:
		return "\033[38;5;196m"
	}
}

// --- Braille canvas ---
//
// Dot layout within a cell (Unicode Braille block U+2800):
//   col 0  col 1
//   bit 0  bit 3   row 0
//   bit 1  bit 4   row 1
//   bit 2  bit 5   row 2
//   bit 6  bit 7   row 3

var brailleBit = [2][4]uint8{
	{0, 1, 2, 6},
	{3, 4, 5, 7},
}

type brailleCanvas struct {
	termW, termH int
	dots         [][]bool   // [pixH][pixW]
	colors       [][]string // [termH][termW] — last color written to cell wins
}

func newBrailleCanvas(termW, termH int) *brailleCanvas {
	pixW, pixH := termW*2, termH*4
	dots := make([][]bool, pixH)
	for i := range dots {
		dots[i] = make([]bool, pixW)
	}
	colors := make([][]string, termH)
	for i := range colors {
		colors[i] = make([]string, termW)
	}
	return &brailleCanvas{termW: termW, termH: termH, dots: dots, colors: colors}
}

func (c *brailleCanvas) set(x, y int, color string) {
	pixW, pixH := c.termW*2, c.termH*4
	if x < 0 || x >= pixW || y < 0 || y >= pixH {
		return
	}
	c.dots[y][x] = true
	cellX, cellY := x/2, y/4
	c.colors[cellY][cellX] = color
}

func (c *brailleCanvas) line(x0, y0, x1, y1 int, color string) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 { sx = -1 }
	if y0 > y1 { sy = -1 }
	err := dx - dy
	for {
		c.set(x0, y0, color)
		if x0 == x1 && y0 == y1 { break }
		e2 := 2 * err
		if e2 > -dy { err -= dy; x0 += sx }
		if e2 < dx  { err += dx; y0 += sy }
	}
}

func (c *brailleCanvas) frame() []string {
	lines := make([]string, c.termH)
	for row := 0; row < c.termH; row++ {
		var sb strings.Builder
		for col := 0; col < c.termW; col++ {
			var bits uint8
			for dc := 0; dc < 2; dc++ {
				for dr := 0; dr < 4; dr++ {
					if c.dots[row*4+dr][col*2+dc] {
						bits |= 1 << brailleBit[dc][dr]
					}
				}
			}
			if bits == 0 {
				sb.WriteRune(' ')
			} else {
				color := c.colors[row][col]
				sb.WriteString(color)
				sb.WriteRune(rune(0x2800 + int(bits)))
				sb.WriteString(resetColor)
			}
		}
		lines[row] = sb.String()
	}
	return lines
}

func abs(x int) int {
	if x < 0 { return -x }
	return x
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/display/... -run TestPrintRoute -v
```

Expected: `PASS`

- [ ] **Step 5: Run full suite**

```bash
make all
```

Expected: all green

- [ ] **Step 6: Commit**

```bash
git add internal/display/route.go internal/display/display_test.go
git commit -m "feat(display): add Braille canvas and PrintRoute with elevation gradient"
```

---

## Task 3: Command — routeCmd and ride.go wiring

**Files:**
- Create: `cmd/route.go`
- Modify: `cmd/ride.go`

- [ ] **Step 1: Create cmd/route.go**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/display"
	"github.com/HGClarke/paceline/internal/store"
	"github.com/spf13/cobra"
)

var (
	routeWidth  int
	routeHeight int
)

var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Render the GPS route as a Braille map in the terminal",
	Args:  cobra.NoArgs,
	RunE:  runRoute,
}

func init() {
	routeCmd.Flags().IntVar(&routeWidth, "width", 78, "character width of the map")
	routeCmd.Flags().IntVar(&routeHeight, "height", 28, "character height of the map")
}

func runRoute(_ *cobra.Command, _ []string) error {
	dbPath, err := store.DefaultPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	pts, err := s.GetGPSPoints(currentRideID)
	if err != nil {
		return err
	}

	if len(pts) == 0 {
		fmt.Fprintf(os.Stderr, "Ride #%d has no GPS data.\n", currentRide.Position)
		return nil
	}

	display.PrintRoute(os.Stdout, pts, routeWidth, routeHeight, currentRide.Filename)
	return nil
}
```

- [ ] **Step 2: Register routeCmd in cmd/ride.go**

In `cmd/ride.go`, in the `init()` function, add after the `rideCmd.AddCommand(powerCurveCmd)` line:

```go
rideCmd.AddCommand(routeCmd)
```

- [ ] **Step 3: Build and smoke-test**

```bash
go build -o paceline . && ./paceline ride 1 route
```

Expected: Braille map rendered for ride #1 (if it has GPS data), or "no GPS data" message.

- [ ] **Step 4: Run full suite**

```bash
make all
```

Expected: all green

- [ ] **Step 5: Commit**

```bash
git add cmd/route.go cmd/ride.go
git commit -m "feat(cmd): add ride route subcommand with Braille GPS map"
```

---

## Task 4: Docs — README and feature roadmap

**Files:**
- Modify: `README.md`
- Modify: `docs/feature-roadmap.md`

- [ ] **Step 1: Add route command section to README.md**

After the `### paceline ride <position> power-curve` section, add:

```markdown
### `paceline ride <position> route`

Render the ride's GPS track as a Braille Unicode map with elevation-based colour gradient.

```bash
paceline ride 3 route

# Control output size (defaults: 78×28)
paceline ride 3 route --width 120 --height 40
```

The gradient shades each segment by altitude — dark green at the lowest point through yellow to red at the highest. Falls back to white if the ride has no altitude data. If the ride has no GPS coordinates, a message is shown instead of a map.

Requires that the ride was imported from a file containing GPS coordinates (`.gpx`, `.tcx`, or a `.fit` with a GPS device). Rides imported from files without GPS data will show "no GPS data".
```

- [ ] **Step 2: Update feature-roadmap.md priority matrix**

Change:
```
| 15 | [Route map in terminal](#15-route-map-in-terminal) | 🔵 Lower | High | ⬜ Not Started |
```
To:
```
| 15 | [Route map in terminal](#15-route-map-in-terminal) | 🔵 Lower | Medium | ✅ Completed |
```

- [ ] **Step 3: Add status note to feature #15 section**

At the bottom of the #15 section body, add:

```markdown
> **Status: Completed.** `ride <id> route` is live. Renders GPS track as Unicode Braille art (2×4 sub-pixels per cell — 8× resolution of block characters). Segments are coloured by altitude using a 5-stop ANSI 256-colour gradient (dark green → bright green → yellow → orange → red). Mercator longitude correction applied for accurate shape at all latitudes. `--width` and `--height` flags control terminal dimensions. Effort revised to Medium: Braille encoding is ~60 lines self-contained; no external library required.
```

- [ ] **Step 4: Commit**

```bash
git add README.md docs/feature-roadmap.md
git commit -m "docs: add route map command to README and mark roadmap #15 completed"
```
