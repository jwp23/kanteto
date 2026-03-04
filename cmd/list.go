package cmd

import (
	"fmt"
	"time"

	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long:  "Show tasks organized by OVERDUE / TODAY / UPCOMING sections.",
	RunE: func(cmd *cobra.Command, args []string) error {
		overdue, err := svc.ListOverdue()
		if err != nil {
			return err
		}

		today, err := svc.ListToday()
		if err != nil {
			return err
		}

		now := time.Now()
		endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		endOfWeek := endOfDay.AddDate(0, 0, 7)
		upcoming, err := svc.ListByDateRange(endOfDay, endOfWeek)
		if err != nil {
			return err
		}

		undated, err := svc.ListUndated()
		if err != nil {
			return err
		}

		if len(overdue) == 0 && len(today) == 0 && len(upcoming) == 0 && len(undated) == 0 {
			fmt.Println("No tasks. Add one with: kt add \"your task\"")
			return nil
		}

		if len(overdue) > 0 {
			fmt.Println("OVERDUE")
			printTasks(overdue)
			fmt.Println()
		}

		if len(today) > 0 {
			fmt.Println("TODAY")
			printTasks(today)
			fmt.Println()
		}

		if len(upcoming) > 0 {
			fmt.Println("UPCOMING")
			printTasks(upcoming)
			fmt.Println()
		}

		if len(undated) > 0 {
			fmt.Println("ANYTIME")
			printTasks(undated)
		}

		return nil
	},
}

func printTasks(tasks []task.Task) {
	for _, t := range tasks {
		id := t.ID[:8]
		if t.DueAt != nil {
			fmt.Printf("  %s  %s  (due %s)\n", id, t.Title, t.DueAt.Format("Mon Jan 2 3:04PM"))
		} else {
			fmt.Printf("  %s  %s\n", id, t.Title)
		}
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
