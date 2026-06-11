package display

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/HGClarke/paceline/internal/store"
)

const resetColor = "\033[0m"

// GPSBounds returns the (minLat, maxLat, minLon, maxLon) bounding box of pts.
// Assumes len(pts) > 0.
func GPSBounds(pts []store.GPSPoint) (minLat, maxLat, minLon, maxLon float64) {
	minLat, maxLat = pts[0].Lat, pts[0].Lat
	minLon, maxLon = pts[0].Lon, pts[0].Lon
	for _, p := range pts[1:] {
		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
		if p.Lon < minLon {
			minLon = p.Lon
		}
		if p.Lon > maxLon {
			maxLon = p.Lon
		}
	}
	return
}

// PrintRoute renders a GPS route as a Braille Unicode art map to w.
// width and height are the terminal character dimensions of the data area.
// filename is shown in the footer.
func PrintRoute(w io.Writer, pts []store.GPSPoint, width, height int, filename string) {
	if len(pts) == 0 {
		fmt.Fprintln(w, "No GPS data available for this ride.")
		return
	}

	pixW := width * 2
	pixH := height * 4

	// Bounding box with 3% padding per side
	minLat, maxLat, minLon, maxLon := GPSBounds(pts)
	minAlt, maxAlt := math.MaxFloat64, -math.MaxFloat64
	hasAlt := false
	for _, p := range pts {
		if p.AltitudeM != nil {
			hasAlt = true
			if *p.AltitudeM < minAlt {
				minAlt = *p.AltitudeM
			}
			if *p.AltitudeM > maxAlt {
				maxAlt = *p.AltitudeM
			}
		}
	}
	latRange := (maxLat - minLat) * 1.06
	lonRange := (maxLon - minLon) * 1.06
	midLat := (minLat + maxLat) / 2
	midLon := (minLon + maxLon) / 2

	altNorm := func(p store.GPSPoint) float64 {
		if !hasAlt || p.AltitudeM == nil || maxAlt == minAlt {
			return 0.5
		}
		return (*p.AltitudeM - minAlt) / (maxAlt - minAlt)
	}

	// Degenerate case: single point or all identical coordinates — can't project.
	if latRange == 0 || lonRange == 0 {
		c := newBrailleCanvas(width, height)
		color := "\033[37m"
		if hasAlt {
			color = altColor(altNorm(pts[0]))
		}
		c.set(pixW/2, pixH/2, color)
		fmt.Fprintln(w, "  ┌"+strings.Repeat("─", width)+"┐")
		for _, row := range c.frame() {
			fmt.Fprintf(w, "  │%s│\n", row)
		}
		fmt.Fprintln(w, "  └"+strings.Repeat("─", width)+"┘")
		fmt.Fprintln(w, "  "+filename)
		return
	}

	// Mercator longitude correction: 1° lon ≠ 1° lat in metres
	cosLat := math.Cos(midLat * math.Pi / 180)

	// Scale to fit pixel canvas preserving aspect ratio.
	// Braille sub-pixels are 2-wide × 4-tall per char cell.
	// Terminal chars are ~2:1 tall:wide, so sub-pixels are physically square — no extra correction needed.
	scaleX := float64(pixW-1) / (lonRange * cosLat)
	scaleY := float64(pixH-1) / latRange
	scale := math.Min(scaleX, scaleY)

	project := func(p store.GPSPoint) (x, y int) {
		x = int(math.Round((p.Lon-midLon)*cosLat*scale)) + pixW/2
		y = int(math.Round(-(p.Lat-midLat)*scale)) + pixH/2
		return
	}

	c := newBrailleCanvas(width, height)
	for i := 1; i < len(pts); i++ {
		x0, y0 := project(pts[i-1])
		x1, y1 := project(pts[i])
		norm := (altNorm(pts[i-1]) + altNorm(pts[i])) / 2
		color := "\033[37m"
		if hasAlt {
			color = altColor(norm)
		}
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

func altColor(norm float64) string {
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

// Braille dot layout within a cell (Unicode Braille block U+2800):
//
//	col 0  col 1
//	bit 0  bit 3   row 0
//	bit 1  bit 4   row 1
//	bit 2  bit 5   row 2
//	bit 6  bit 7   row 3

var brailleBit = [2][4]uint8{
	{0, 1, 2, 6},
	{3, 4, 5, 7},
}

type brailleCanvas struct {
	termW, termH int
	dots         [][]bool
	colors       [][]string // per cell; last colour drawn into cell wins
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
	c.colors[y/4][x/2] = color
}

func (c *brailleCanvas) line(x0, y0, x1, y1 int, color string) {
	dx := routeAbs(x1 - x0)
	dy := routeAbs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		c.set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
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

func routeAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
