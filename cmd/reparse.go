package cmd

import (
	"fmt"

	"github.com/jwp23/kanteto/internal/nlp"
	"github.com/spf13/cobra"
)

var reparseApply bool

var reparseCmd = &cobra.Command{
	Use:   "reparse",
	Short: "Re-extract deadlines from undated task titles",
	Long: `Scans all tasks without a due date and re-runs deadline extraction on their titles.
By default shows proposed changes (dry run). Use --apply to save.`,
	Example: `  kt reparse
  kt reparse --apply`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, err := svc.ListUndated()
		if err != nil {
			return fmt.Errorf("list undated: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No undated tasks found.")
			return nil
		}

		var matched int
		for _, tk := range tasks {
			title, dueAt := nlp.ExtractDeadline(tk.Title)
			if dueAt == nil {
				continue
			}

			matched++
			fmt.Fprintf(cmd.OutOrStdout(), "  %s  %q -> %q  (due %s)\n",
				tk.ID[:8], tk.Title, title, dueAt.Format("Mon Jan 2 3:04PM"))

			if reparseApply {
				tk.Title = title
				tk.DueAt = dueAt
				if err := svc.Update(tk); err != nil {
					return fmt.Errorf("update %s: %w", tk.ID[:8], err)
				}
				if err := svc.SetDueAt(tk.ID, *dueAt); err != nil {
					return fmt.Errorf("set deadline %s: %w", tk.ID[:8], err)
				}
			}
		}

		if matched == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No deadlines detected in undated task titles.")
			return nil
		}

		if !reparseApply {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d task(s) would be updated. Run with --apply to save.\n", matched)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d task(s) updated.\n", matched)
		}
		return nil
	},
}

func init() {
	reparseCmd.Flags().BoolVar(&reparseApply, "apply", false, "apply changes (default is dry run)")
	rootCmd.AddCommand(reparseCmd)
}
