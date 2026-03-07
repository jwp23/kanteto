package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short:   "Delete a task permanently",
	Example: `  kt rm abc1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		tk, err := svc.Get(id)
		if err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		if err := svc.Delete(id); err != nil {
			return err
		}

		fmt.Printf("Deleted: %s\n", tk.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
