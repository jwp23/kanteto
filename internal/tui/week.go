package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func renderWeekView(m model) string {
	var b strings.Builder

	// Find the start of the week (Sunday)
	date := m.viewDate
	weekday := date.Weekday()
	startOfWeek := date.AddDate(0, 0, -int(weekday))

	header := fmt.Sprintf("  Week of %s", startOfWeek.Format("January 2, 2006"))
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	now := time.Now()
	colWidth := 20
	if m.width > 0 {
		colWidth = max((m.width-4)/7, 12)
	}

	// Day headers
	var headers []string
	for i := 0; i < 7; i++ {
		d := startOfWeek.AddDate(0, 0, i)
		label := d.Format("Mon 1/2")
		style := lipgloss.NewStyle().Width(colWidth).Bold(true)
		if d.Year() == now.Year() && d.YearDay() == now.YearDay() {
			style = style.Foreground(lipgloss.Color("14"))
		}
		headers = append(headers, style.Render(label))
	}
	b.WriteString("  " + strings.Join(headers, " "))
	b.WriteString("\n")

	// Get tasks for each day
	var columns [][]string
	maxRows := 0
	for i := 0; i < 7; i++ {
		d := startOfWeek.AddDate(0, 0, i)
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		dayEnd := dayStart.Add(24 * time.Hour)

		tasks, _ := m.svc.ListByDateRange(dayStart, dayEnd)
		var lines []string
		for _, t := range tasks {
			color := UrgencyColor(t.DueAt)
			style := lipgloss.NewStyle().Foreground(color).Width(colWidth)
			timeStr := ""
			if t.DueAt != nil {
				timeStr = " " + t.DueAt.Format("3PM")
			}
			title := t.Title
			if len(title) > colWidth-6 {
				title = title[:colWidth-6] + ".."
			}
			lines = append(lines, style.Render(title+timeStr))
		}
		columns = append(columns, lines)
		if len(lines) > maxRows {
			maxRows = len(lines)
		}
	}

	// Render rows
	emptyCell := lipgloss.NewStyle().Width(colWidth).Render("")
	for row := 0; row < maxRows; row++ {
		var cells []string
		for col := 0; col < 7; col++ {
			if row < len(columns[col]) {
				cells = append(cells, columns[col][row])
			} else {
				cells = append(cells, emptyCell)
			}
		}
		b.WriteString("  " + strings.Join(cells, " "))
		b.WriteString("\n")
	}

	if maxRows == 0 {
		b.WriteString(dimStyle.Render("\n  No tasks this week."))
		b.WriteString("\n")
	}

	return b.String()
}
