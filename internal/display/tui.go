package display

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hollandclarke/paceline/internal/parser"
	"golang.org/x/term"
)

// IsTTY reports whether stdout is an interactive terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

type ridesModel struct {
	rides    []parser.Ride
	cursor   int
	page     int
	limit    int
	total    int
	selected *parser.Ride
	err      error
	loadPage func(page int) ([]parser.Ride, int, error)
}

func (m ridesModel) Init() tea.Cmd { return nil }

func (m ridesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rides)-1 {
				m.cursor++
			}
		case "n", "right":
			maxPage := 1
			if m.limit > 0 {
				maxPage = (m.total + m.limit - 1) / m.limit
			}
			if m.page < maxPage {
				m.page++
				rides, total, err := m.loadPage(m.page)
				if err != nil {
					m.err = err
				} else {
					m.rides = rides
					m.total = total
				}
				m.cursor = 0
			}
		case "p", "left":
			if m.page > 1 {
				m.page--
				rides, total, err := m.loadPage(m.page)
				if err != nil {
					m.err = err
				} else {
					m.rides = rides
					m.total = total
				}
				m.cursor = 0
			}
		case "enter":
			if len(m.rides) > 0 {
				r := m.rides[m.cursor]
				m.selected = &r
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m ridesModel) View() string {
	var sb strings.Builder
	if m.err != nil {
		fmt.Fprintf(&sb, "Error: %v\n\n", m.err)
	}
	maxPage := 1
	if m.limit > 0 {
		maxPage = (m.total + m.limit - 1) / m.limit
	}
	if maxPage < 1 {
		maxPage = 1
	}
	fmt.Fprintf(&sb, "Rides — Page %d of %d  (↑/↓ move, n/p page, enter select, q quit)\n\n", m.page, maxPage)

	header := fmt.Sprintf("  %-4s  %-12s  %-10s  %-10s  %-10s\n", "ID", "Date", "Distance", "Duration", "Elevation")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("─", 60) + "\n")

	for i, r := range m.rides {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		fmt.Fprintf(&sb, "%s%-4d  %-12s  %-10s  %-10s  %-10s\n",
			cursor,
			r.ID,
			r.RecordedAt.Format("2006-01-02"),
			fmt.Sprintf("%.1f km", r.DistanceM/1000),
			formatDuration(r.DurationS),
			fmt.Sprintf("%.0f m", r.ElevationGainM),
		)
	}
	return sb.String()
}

// RunRidesTUI launches the interactive rides list.
// loadPage is called when the user navigates to a different page.
// If the user selects a ride, the selected Ride is returned; otherwise nil.
func RunRidesTUI(w io.Writer, initialRides []parser.Ride, total, limit int, loadPage func(page int) ([]parser.Ride, int, error)) (*parser.Ride, error) {
	m := ridesModel{
		rides:    initialRides,
		total:    total,
		page:     1,
		limit:    limit,
		loadPage: loadPage,
	}
	p := tea.NewProgram(m, tea.WithOutput(w))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(ridesModel)
	return final.selected, nil
}
