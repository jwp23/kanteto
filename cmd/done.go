package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a task as completed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		tk, err := svc.Get(id)
		if err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		if err := svc.Complete(id); err != nil {
			return err
		}

		fmt.Printf("Completed: %s\n", tk.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
