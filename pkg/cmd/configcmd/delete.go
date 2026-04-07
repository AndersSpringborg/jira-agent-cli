package configcmd

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		profile string
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("deletion requires --yes")
			}

			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			if config.DeleteProfile(cfg, profile) {
				if err := config.Save(cfg); err != nil {
					return err
				}
				return driver.Message("Profile '%s' deleted.", profile)
			}
			return driver.Message("Profile '%s' not found.", profile)
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile name")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm deletion")
	_ = cmd.MarkFlagRequired("profile")

	return cmd
}
