package cmd

import (
	"fmt"
	"time"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"
)

var (
	editTitle string
	editBy    string
	editEvery string
)

var editCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit a task's title, deadline, or recurrence",
	Example: `  kt edit abc1 --title "New title"
  kt edit abc1 --by "friday at 3pm"
  kt edit abc1 --every "weekdays at 9am"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		tk, err := svc.Get(id)
		if err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		if editTitle != "" {
			tk.Title = editTitle
		}

		if editBy != "" {
			t, err := nlp.ParseDate(editBy)
			if err != nil {
				return fmt.Errorf("parse deadline: %w", err)
			}
			tk.DueAt = &t
		}

		if editEvery != "" {
			pattern, timeStr, err := task.ParseRecurrenceSpec(editEvery)
			if err != nil {
				return fmt.Errorf("parse recurrence: %w", err)
			}
			tk.RecurrencePattern = pattern
			tk.RecurrenceTime = timeStr

			nextDue, err := task.NextOccurrence(pattern, timeStr, time.Now())
			if err != nil {
				return fmt.Errorf("compute next occurrence: %w", err)
			}
			tk.DueAt = &nextDue
		}

		if err := svc.Update(tk); err != nil {
			return err
		}

		fmt.Printf("Updated: %s", tk.Title)
		if tk.DueAt != nil {
			fmt.Printf(" (due %s)", tk.DueAt.Format("Mon Jan 2 3:04PM"))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "new title for the task")
	editCmd.Flags().StringVar(&editBy, "by", "", "new deadline in natural language")
	editCmd.Flags().StringVar(&editEvery, "every", "", "new recurrence pattern")
	rootCmd.AddCommand(editCmd)
}
