package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"
)

var (
	addBy   string
	addEvery string
	addTags []string
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Long: `Add a task with an optional natural language deadline or recurrence.
  kt add "Call dentist" --by "march 11"
  kt add "Send weekly update" --every "weekdays at 4pm"`,
	Example: `  kt add "Call dentist" --by "march 11"
  kt add "Send weekly update" --every "weekdays at 4pm"
  kt add "Buy groceries"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		if addEvery != "" {
			return addRecurring(title, addEvery)
		}

		var dueAt *time.Time
		if addBy != "" {
			t, err := nlp.ParseDate(addBy)
			if err != nil {
				return fmt.Errorf("parse deadline: %w", err)
			}
			dueAt = &t
		}

		tk, err := svc.Add(title, dueAt, addTags...)
		if err != nil {
			return err
		}

		if tk.DueAt != nil {
			fmt.Printf("Added: %s (due %s) [%s]", tk.Title, tk.DueAt.Format("Mon Jan 2 3:04PM"), tk.ID[:8])
		} else {
			fmt.Printf("Added: %s [%s]", tk.Title, tk.ID[:8])
		}
		if len(tk.Tags) > 0 {
			fmt.Printf(" %s", formatTags(tk.Tags))
		}
		fmt.Println()
		return nil
	},
}

func addRecurring(title, every string) error {
	pattern, timeStr, err := task.ParseRecurrenceSpec(every)
	if err != nil {
		return fmt.Errorf("parse recurrence: %w", err)
	}

	tk, err := svc.AddRecurring(title, pattern, timeStr)
	if err != nil {
		return err
	}

	fmt.Printf("Added: %s (recurring %s at %s, next %s) [%s]\n",
		tk.Title, pattern, timeStr,
		tk.DueAt.Format("Mon Jan 2 3:04PM"), tk.ID[:8])
	return nil
}

func formatTags(tags []string) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = "[" + t + "]"
	}
	return strings.Join(parts, " ")
}

func init() {
	addCmd.Flags().StringVar(&addBy, "by", "", "deadline in natural language (e.g. \"march 11\", \"tomorrow at 3pm\")")
	addCmd.Flags().StringVar(&addEvery, "every", "", "recurrence pattern (e.g. \"weekdays at 4pm\", \"friday at 5pm\")")
	addCmd.Flags().StringArrayVar(&addTags, "tag", nil, "tag for the task (can be repeated)")
	rootCmd.AddCommand(addCmd)
}
