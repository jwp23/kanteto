package cmd

import (
	"fmt"
	"time"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/spf13/cobra"
)

var addBy string

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Long:  `Add a task with an optional natural language deadline. Example: kt add "Call dentist" --by "march 11"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		var dueAt *time.Time
		if addBy != "" {
			t, err := nlp.ParseDate(addBy)
			if err != nil {
				return fmt.Errorf("parse deadline: %w", err)
			}
			dueAt = &t
		}

		tk, err := svc.Add(title, dueAt)
		if err != nil {
			return err
		}

		if tk.DueAt != nil {
			fmt.Printf("Added: %s (due %s) [%s]\n", tk.Title, tk.DueAt.Format("Mon Jan 2 3:04PM"), tk.ID[:8])
		} else {
			fmt.Printf("Added: %s [%s]\n", tk.Title, tk.ID[:8])
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addBy, "by", "", "deadline in natural language (e.g. \"march 11\", \"tomorrow at 3pm\")")
	rootCmd.AddCommand(addCmd)
}
