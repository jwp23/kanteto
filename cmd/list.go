package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/jwp23/kanteto/internal/task"
	"github.com/spf13/cobra"
)

var (
	listNext bool
	listPrev bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: "Show tasks organized by OVERDUE / TODAY / UPCOMING sections.",
	Example: `  kt list
  kt list --next
  kt list --prev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listNext && listPrev {
			return fmt.Errorf("cannot use --next and --prev together")
		}

		w := cmd.OutOrStdout()

		refDate := time.Now()
		if listNext {
			refDate = refDate.AddDate(0, 0, 1)
		} else if listPrev {
			refDate = refDate.AddDate(0, 0, -1)
		}

		startOfDay := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, refDate.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		// Overdue: tasks due before start of the reference day
		overdue, err := svc.ListOverdueAsOf(startOfDay)
		if err != nil {
			return err
		}

		today, err := svc.ListByDateRange(startOfDay, endOfDay)
		if err != nil {
			return err
		}

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
			fmt.Fprintln(w, "No tasks. Add one with: kt add \"your task\"")
			return nil
		}

		if len(overdue) > 0 {
			fmt.Fprintln(w, "OVERDUE")
			fprintTasks(w, overdue)
			fmt.Fprintln(w)
		}

		if len(today) > 0 {
			fmt.Fprintln(w, "TODAY")
			fprintTasks(w, today)
			fmt.Fprintln(w)
		}

		if len(upcoming) > 0 {
			fmt.Fprintln(w, "UPCOMING")
			fprintTasks(w, upcoming)
			fmt.Fprintln(w)
		}

		if len(undated) > 0 {
			fmt.Fprintln(w, "ANYTIME")
			fprintTasks(w, undated)
		}

		return nil
	},
}

func fprintTasks(w io.Writer, tasks []task.Task) {
	for _, t := range tasks {
		id := t.ID[:8]
		if t.DueAt != nil {
			fmt.Fprintf(w, "  %s  %s  (due %s)\n", id, t.Title, t.DueAt.Format("Mon Jan 2 3:04PM"))
		} else {
			fmt.Fprintf(w, "  %s  %s\n", id, t.Title)
		}
	}
}

func init() {
	listCmd.Flags().BoolVar(&listNext, "next", false, "Show next day's tasks")
	listCmd.Flags().BoolVar(&listPrev, "prev", false, "Show previous day's tasks")
	rootCmd.AddCommand(listCmd)
}
