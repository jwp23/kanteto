package nlp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	monthNames = map[string]time.Month{
		"january": time.January, "february": time.February, "march": time.March,
		"april": time.April, "may": time.May, "june": time.June,
		"july": time.July, "august": time.August, "september": time.September,
		"october": time.October, "november": time.November, "december": time.December,
		"jan": time.January, "feb": time.February, "mar": time.March,
		"apr": time.April, "jun": time.June, "jul": time.July,
		"aug": time.August, "sep": time.September, "oct": time.October,
		"nov": time.November, "dec": time.December,
	}

	dayNames = map[string]time.Weekday{
		"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
		"wednesday": time.Wednesday, "thursday": time.Thursday,
		"friday": time.Friday, "saturday": time.Saturday,
		"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
		"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday,
		"sat": time.Saturday,
	}

	timeRe    = regexp.MustCompile(`(?i)at\s+(\d{1,2})(:\d{2})?\s*(am|pm)?`)
	isoDateRe = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	slashRe   = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})$`)
)

// ParseDate parses a natural language date string relative to time.Now().
func ParseDate(s string) (time.Time, error) {
	return ParseDateRelativeTo(s, time.Now())
}

// ParseDateRelativeTo parses a natural language date string relative to a given time.
func ParseDateRelativeTo(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	// "in 5 minutes", "in 1 hour", "in 2 hours", "in 30m"
	if strings.HasPrefix(lower, "in ") {
		durStr := strings.TrimPrefix(lower, "in ")
		d, err := ParseDuration(durStr)
		if err == nil {
			return now.Add(d), nil
		}
	}

	hour, min := 23, 59
	if m := timeRe.FindStringSubmatch(lower); m != nil {
		h, _ := strconv.Atoi(m[1])
		mn := 0
		if m[2] != "" {
			mn, _ = strconv.Atoi(m[2][1:])
		}
		if strings.EqualFold(m[3], "pm") && h < 12 {
			h += 12
		} else if strings.EqualFold(m[3], "am") && h == 12 {
			h = 0
		}
		hour, min = h, mn
		// Strip the time part for date parsing
		lower = strings.TrimSpace(timeRe.ReplaceAllString(lower, ""))

		// If only a time was given (e.g., "at 3pm"), treat as today
		if lower == "" {
			return time.Date(now.Year(), now.Month(), now.Day(), h, mn, 0, 0, now.Location()), nil
		}
	}

	// ISO date: 2026-03-15
	if m := isoDateRe.FindStringSubmatch(lower); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		return time.Date(y, time.Month(mo), d, hour, min, 0, 0, now.Location()), nil
	}

	// Slash date: 3/15
	if m := slashRe.FindStringSubmatch(lower); m != nil {
		mo, _ := strconv.Atoi(m[1])
		d, _ := strconv.Atoi(m[2])
		y := now.Year()
		candidate := time.Date(y, time.Month(mo), d, hour, min, 0, 0, now.Location())
		if candidate.Before(now) {
			candidate = time.Date(y+1, time.Month(mo), d, hour, min, 0, 0, now.Location())
		}
		return candidate, nil
	}

	// "today"
	if lower == "today" {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location()), nil
	}

	// "tomorrow"
	if lower == "tomorrow" {
		tm := now.AddDate(0, 0, 1)
		return time.Date(tm.Year(), tm.Month(), tm.Day(), hour, min, 0, 0, now.Location()), nil
	}

	// "next <weekday>"
	if strings.HasPrefix(lower, "next ") {
		dayStr := strings.TrimPrefix(lower, "next ")
		if wd, ok := dayNames[dayStr]; ok {
			return nextWeekday(now, wd, hour, min), nil
		}
	}

	// "<month> <day>"
	if t, ok := parseMonthDay(lower, now, hour, min); ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %q", s)
}

// ParseDuration parses natural language durations like "1 hour", "30 minutes", "2h".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Try Go's built-in parser first (handles "30m", "2h", "1h30m")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Natural language: "<n> <unit>"
	re := regexp.MustCompile(`^(\d+)\s*(hours?|minutes?|mins?|days?|d)$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("unable to parse duration: %q", s)
	}

	n, _ := strconv.Atoi(m[1])
	unit := m[2]

	switch {
	case strings.HasPrefix(unit, "hour"):
		return time.Duration(n) * time.Hour, nil
	case strings.HasPrefix(unit, "min"):
		return time.Duration(n) * time.Minute, nil
	case strings.HasPrefix(unit, "day"), unit == "d":
		return time.Duration(n) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("unable to parse duration: %q", s)
}

func nextWeekday(now time.Time, wd time.Weekday, hour, min int) time.Time {
	daysAhead := int(wd) - int(now.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	d := now.AddDate(0, 0, daysAhead)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, min, 0, 0, now.Location())
}

func parseMonthDay(s string, now time.Time, hour, min int) (time.Time, bool) {
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return time.Time{}, false
	}

	month, ok := monthNames[parts[0]]
	if !ok {
		return time.Time{}, false
	}

	day, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, false
	}

	y := now.Year()
	candidate := time.Date(y, month, day, hour, min, 0, 0, now.Location())
	if candidate.Before(now) {
		candidate = time.Date(y+1, month, day, hour, min, 0, 0, now.Location())
	}
	return candidate, true
}

// deadlineMarker defines a split point in inline input.
// keyword is the part that belongs to the date expression (e.g., "in", "at").
type deadlineMarker struct {
	sep     string // the separator to search for in input
	keyword string // prefix to prepend to the date string (may be empty)
}

var deadlineMarkers = []deadlineMarker{
	{" in ", "in "},
	{" by ", ""},
	{" at ", "at "},
	{" --by ", ""},
}

// ExtractDeadline splits inline input like "test kt in 5 minutes" into
// a title ("test kt") and a parsed deadline. If no deadline is found,
// returns the full input as the title with a nil time.
func ExtractDeadline(input string) (title string, dueAt *time.Time) {
	lower := strings.ToLower(input)

	for _, m := range deadlineMarkers {
		idx := strings.LastIndex(lower, m.sep)
		if idx < 0 {
			continue
		}

		candidateTitle := strings.TrimSpace(input[:idx])
		candidateDate := m.keyword + strings.TrimSpace(input[idx+len(m.sep):])

		if candidateTitle == "" || candidateDate == "" {
			continue
		}

		t, err := ParseDate(candidateDate)
		if err == nil {
			return candidateTitle, &t
		}
	}

	return input, nil
}
