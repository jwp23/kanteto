package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func renderMonthView(m model) string {
	var b strings.Builder

	date := m.viewDate
	year, month := date.Year(), date.Month()
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, date.Location())
	now := time.Now()

	header := fmt.Sprintf("  %s %d", month, year)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	colWidth := 6
	if m.width > 0 {
		colWidth = max((m.width-4)/7, 5)
	}

	// Weekday headers
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	var headerCells []string
	for _, name := range dayNames {
		style := lipgloss.NewStyle().Width(colWidth).Bold(true).Foreground(lipgloss.Color("7"))
		headerCells = append(headerCells, style.Render(name))
	}
	b.WriteString("  " + strings.Join(headerCells, " "))
	b.WriteString("\n")

	// Calendar grid
	startWeekday := firstOfMonth.Weekday()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, date.Location()).Day()

	day := 1
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}

		var cells []string
		for wd := 0; wd < 7; wd++ {
			if (week == 0 && wd < int(startWeekday)) || day > daysInMonth {
				cells = append(cells, lipgloss.NewStyle().Width(colWidth).Render(""))
				continue
			}

			d := time.Date(year, month, day, 0, 0, 0, 0, date.Location())
			tasks := m.monthTasks[day]

			label := fmt.Sprintf("%d", day)
			if len(tasks) > 0 {
				label = fmt.Sprintf("%d(%d)", day, len(tasks))
			}

			selected := day == m.monthCursorDay
			if selected {
				label = fmt.Sprintf("[%s]", label)
			}

			style := lipgloss.NewStyle().Width(colWidth)
			if selected {
				style = style.Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("236"))
			} else if d.Year() == now.Year() && d.YearDay() == now.YearDay() {
				style = style.Bold(true).Foreground(lipgloss.Color("14"))
			} else if len(tasks) > 0 {
				style = style.Foreground(lipgloss.Color("11"))
			}

			cells = append(cells, style.Render(label))
			day++
		}
		b.WriteString("  " + strings.Join(cells, " "))
		b.WriteString("\n")
	}

	return b.String()
}
