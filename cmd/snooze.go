package cmd

import (
	"fmt"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/spf13/cobra"
)

var snoozeFor string

var snoozeCmd = &cobra.Command{
	Use:   "snooze [id]",
	Short: "Postpone a task's deadline",
	Long: `Snooze a task by a duration. Example: kt snooze abc123 --for "1 hour"`,
	Example: `  kt snooze abc1 --for "1 hour"
  kt snooze abc1 --for 30m`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		d, err := nlp.ParseDuration(snoozeFor)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}

		if err := svc.Snooze(id, d); err != nil {
			return err
		}

		tk, err := svc.Get(id)
		if err != nil {
			return err
		}

		fmt.Printf("Snoozed: %s (now due %s)\n", tk.Title, tk.DueAt.Format("Mon Jan 2 3:04PM"))
		return nil
	},
}

func init() {
	snoozeCmd.Flags().StringVar(&snoozeFor, "for", "1h", "duration to snooze (e.g. \"1 hour\", \"30m\")")
	rootCmd.AddCommand(snoozeCmd)
}
