package cmd

import (
	"fmt"
	"strings"
)

// resolveID finds a task ID from a prefix. Users can type short prefixes
// (e.g., "abc1") and we match against all tasks.
func resolveID(prefix string) (string, error) {
	tasks, err := svc.ListAll()
	if err != nil {
		return "", err
	}

	var matches []string
	for _, t := range tasks {
		if strings.HasPrefix(t.ID, prefix) {
			matches = append(matches, t.ID)
		}
	}

	if len(matches) == 0 {
		// Try as exact ID (may be a completed task)
		if _, err := svc.Get(prefix); err == nil {
			return prefix, nil
		}
		return "", fmt.Errorf("no task found matching %q — run 'kt list' to see available tasks", prefix)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("prefix %q matches %d tasks — try more characters to narrow it down", prefix, len(matches))
	}

	return matches[0], nil
}
