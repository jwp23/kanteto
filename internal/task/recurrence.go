package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var dayNameToWeekday = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday,
	"friday": time.Friday, "saturday": time.Saturday,
}

// ParseRecurrenceSpec parses a natural language recurrence string like
// "weekdays at 4pm" into a pattern and time component.
func ParseRecurrenceSpec(s string) (pattern, timeStr string, err error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Strip "every " prefix
	s = strings.TrimPrefix(s, "every ")

	// Replace "day" with "daily" for "every day at ..." form
	if strings.HasPrefix(s, "day ") {
		s = "daily " + s[4:]
	}

	// Split on " at "
	parts := strings.SplitN(s, " at ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("recurrence must contain 'at <time>': %q", s)
	}

	pattern = strings.TrimSpace(parts[0])
	timeStr = strings.TrimSpace(parts[1])

	// Validate pattern
	switch pattern {
	case "daily", "weekly", "weekdays":
		// valid
	default:
		if _, ok := dayNameToWeekday[pattern]; !ok {
			return "", "", fmt.Errorf("unknown recurrence pattern: %q", pattern)
		}
	}

	return pattern, timeStr, nil
}

// NextOccurrence computes the next occurrence of a recurring task after from.
func NextOccurrence(pattern, timeStr string, from time.Time) (time.Time, error) {
	h, m, err := parseTimeStr(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	pattern = strings.ToLower(pattern)

	switch pattern {
	case "daily":
		next := time.Date(from.Year(), from.Month(), from.Day()+1, h, m, 0, 0, from.Location())
		return next, nil

	case "weekly":
		next := from.AddDate(0, 0, 7)
		return time.Date(next.Year(), next.Month(), next.Day(), h, m, 0, 0, from.Location()), nil

	case "weekdays":
		next := from
		for {
			next = next.AddDate(0, 0, 1)
			wd := next.Weekday()
			if wd != time.Saturday && wd != time.Sunday {
				return time.Date(next.Year(), next.Month(), next.Day(), h, m, 0, 0, from.Location()), nil
			}
		}

	default:
		// Specific day name
		wd, ok := dayNameToWeekday[pattern]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown recurrence pattern: %q", pattern)
		}
		daysAhead := int(wd) - int(from.Weekday())
		if daysAhead <= 0 {
			daysAhead += 7
		}
		next := from.AddDate(0, 0, daysAhead)
		return time.Date(next.Year(), next.Month(), next.Day(), h, m, 0, 0, from.Location()), nil
	}
}

func parseTimeStr(s string) (hour, min int, err error) {
	s = strings.TrimSpace(strings.ToLower(s))

	isPM := strings.HasSuffix(s, "pm")
	isAM := strings.HasSuffix(s, "am")
	if isPM || isAM {
		s = strings.TrimSuffix(s, "pm")
		s = strings.TrimSuffix(s, "am")
	}

	parts := strings.SplitN(s, ":", 2)
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid time: %q", s)
	}

	m := 0
	if len(parts) == 2 {
		m, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid time minutes: %q", s)
		}
	}

	if isPM && h < 12 {
		h += 12
	} else if isAM && h == 12 {
		h = 0
	}

	return h, m, nil
}
