package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwp23/kanteto/internal/task"
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

	// Single query for the whole week, then bucket by day
	weekStart := time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
	weekEnd := weekStart.AddDate(0, 0, 7)
	allTasks, _ := m.svc.ListByDateRange(weekStart, weekEnd)

	tasksByDay := make(map[string][]task.Task)
	for _, t := range allTasks {
		if t.DueAt != nil {
			key := t.DueAt.Format("2006-01-02")
			tasksByDay[key] = append(tasksByDay[key], t)
		}
	}

	// Build columns from bucketed tasks
	var columns [][]string
	maxRows := 0
	for i := 0; i < 7; i++ {
		d := startOfWeek.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		tasks := tasksByDay[key]
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
