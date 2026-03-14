package cmd

import (
	"fmt"

	"github.com/jwp23/kanteto/internal/config"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage task profiles",
	Long:  "Switch between profiles to scope task views (e.g., work vs personal).",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := rawStore.ListProfiles()
		if err != nil {
			return err
		}
		active := activeProfile()
		if len(profiles) == 0 {
			fmt.Println("No profiles yet. Tasks use the 'default' profile.")
			return nil
		}
		for _, p := range profiles {
			marker := "  "
			if p == active {
				marker = "* "
			}
			fmt.Printf("%s%s\n", marker, p)
		}
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(activeProfile())
		return nil
	},
}

var profileUseCmd = &cobra.Command{
	Use:     "use [name]",
	Short:   "Switch to a profile",
	Example: `  kt profile use work`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg.ActiveProfile = name
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Switched to profile: %s\n", name)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileUseCmd)
	rootCmd.AddCommand(profileCmd)
}
