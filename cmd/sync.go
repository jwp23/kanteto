package cmd

import (
	"fmt"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/jwp23/kanteto/internal/sync"
	"github.com/spf13/cobra"
)

func newSync() *sync.Sync {
	return sync.New(config.DataDir())
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tasks to a remote",
	Long:  "Push and pull tasks to a Dolt remote (GitHub, DoltHub, etc.).",
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Commit and push tasks to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newSync()
		if err := s.Push(); err != nil {
			return err
		}
		fmt.Println("Pushed to remote.")
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull tasks from remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newSync()
		if err := s.Pull(); err != nil {
			return err
		}
		fmt.Println("Pulled from remote.")
		return nil
	},
}

var syncRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage sync remotes",
}

var syncRemoteAddCmd = &cobra.Command{
	Use:     "add [name] [url]",
	Short:   "Add a remote",
	Example: `  kt sync remote add origin https://doltremoteapi.dolthub.com/user/tasks`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newSync()
		if err := s.AddRemote(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Added remote %q → %s\n", args[0], args[1])
		return nil
	},
}

var syncRemoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured remotes",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newSync()
		remotes, err := s.ListRemotes()
		if err != nil {
			return err
		}
		if len(remotes) == 0 {
			fmt.Println("No remotes configured.")
			return nil
		}
		for _, r := range remotes {
			fmt.Println(r)
		}
		return nil
	},
}

func init() {
	syncRemoteCmd.AddCommand(syncRemoteAddCmd)
	syncRemoteCmd.AddCommand(syncRemoteListCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncRemoteCmd)
	rootCmd.AddCommand(syncCmd)
}
