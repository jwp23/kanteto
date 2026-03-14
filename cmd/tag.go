package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:     "tag [id] [tag]",
	Short:   "Add a tag to a task",
	Example: `  kt tag abc1 work`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		if err := svc.AddTag(id, args[1]); err != nil {
			return err
		}

		tk, _ := svc.Get(id)
		fmt.Printf("Tagged: %s %s\n", tk.Title, formatTags(tk.Tags))
		return nil
	},
}

var untagCmd = &cobra.Command{
	Use:     "untag [id] [tag]",
	Short:   "Remove a tag from a task",
	Example: `  kt untag abc1 work`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveID(args[0])
		if err != nil {
			return err
		}

		if err := svc.RemoveTag(id, args[1]); err != nil {
			return err
		}

		tk, _ := svc.Get(id)
		fmt.Printf("Untagged: %s %s\n", tk.Title, formatTags(tk.Tags))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(untagCmd)
}
