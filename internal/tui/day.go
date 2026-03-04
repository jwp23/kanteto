package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwp23/kanteto/internal/task"
)

func renderDayView(m model) string {
	var b strings.Builder

	date := m.viewDate
	header := fmt.Sprintf("  %s  —  Day View", date.Format("Monday, January 2, 2006"))
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	globalIdx := 0

	if len(m.overdue) > 0 {
		b.WriteString(overdueSectionStyle.Render("  OVERDUE"))
		b.WriteString("\n")
		for _, t := range m.overdue {
			b.WriteString(renderTask(t, globalIdx, m.cursor))
			b.WriteString("\n")
			globalIdx++
		}
	}

	if len(m.today) > 0 {
		b.WriteString(todaySectionStyle.Render("  TODAY"))
		b.WriteString("\n")
		for _, t := range m.today {
			b.WriteString(renderTask(t, globalIdx, m.cursor))
			b.WriteString("\n")
			globalIdx++
		}
	}

	if len(m.upcoming) > 0 {
		b.WriteString(upcomingSectionStyle.Render("  UPCOMING"))
		b.WriteString("\n")
		for _, t := range m.upcoming {
			b.WriteString(renderTask(t, globalIdx, m.cursor))
			b.WriteString("\n")
			globalIdx++
		}
	}

	if len(m.undated) > 0 {
		b.WriteString(sectionStyle.Foreground(lipgloss.Color("8")).Render("  ANYTIME"))
		b.WriteString("\n")
		for _, t := range m.undated {
			b.WriteString(renderTask(t, globalIdx, m.cursor))
			b.WriteString("\n")
			globalIdx++
		}
	}

	if globalIdx == 0 {
		b.WriteString(dimStyle.Render("\n  No tasks. Press 'a' to add one."))
		b.WriteString("\n")
	}

	return b.String()
}

func renderTask(t task.Task, idx, cursor int) string {
	prefix := "  "
	if idx == cursor {
		prefix = "> "
	}

	color := UrgencyColor(t.DueAt)
	style := lipgloss.NewStyle().Foreground(color)

	id := t.ID[:8]
	var line string
	if t.DueAt != nil {
		timeStr := t.DueAt.Format("3:04PM")
		if !isToday(*t.DueAt) {
			timeStr = t.DueAt.Format("Jan 2 3:04PM")
		}
		line = fmt.Sprintf("%s%s  %s  %s", prefix, dimStyle.Render(id), style.Render(t.Title), dimStyle.Render(timeStr))
	} else {
		line = fmt.Sprintf("%s%s  %s", prefix, dimStyle.Render(id), style.Render(t.Title))
	}

	if idx == cursor {
		return selectedStyle.Render(line)
	}
	return line
}

func isToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}
