package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Base styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			MarginTop(1)

	overdueSectionStyle = sectionStyle.Foreground(lipgloss.Color("9"))
	todaySectionStyle   = sectionStyle.Foreground(lipgloss.Color("14"))
	upcomingSectionStyle = sectionStyle.Foreground(lipgloss.Color("7"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("15"))

	completedStyle = lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.Color("240"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))

	// Urgency colors: white -> yellow -> amber -> orange -> red
	urgencyWhite  = lipgloss.Color("15")
	urgencyYellow = lipgloss.Color("11")
	urgencyAmber  = lipgloss.Color("214")
	urgencyOrange = lipgloss.Color("208")
	urgencyRed    = lipgloss.Color("9")
)

// UrgencyColor returns a color based on how close the deadline is.
func UrgencyColor(dueAt *time.Time) lipgloss.Color {
	if dueAt == nil {
		return urgencyWhite
	}

	until := time.Until(*dueAt)

	switch {
	case until < 0:
		return urgencyRed
	case until < 30*time.Minute:
		return urgencyOrange
	case until < time.Hour:
		return urgencyAmber
	case until < 2*time.Hour:
		return urgencyYellow
	default:
		return urgencyWhite
	}
}
