package configcmd

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newUseCmd(f *cmdutil.Factory) *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "use",
		Short: "Set the default profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			if _, ok := cfg.Profiles[profile]; !ok {
				return fmt.Errorf("profile '%s' not found", profile)
			}

			cfg.DefaultProfile = profile
			if err := config.Save(cfg); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Default profile set to '%s'.", profile)
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile name")
	_ = cmd.MarkFlagRequired("profile")

	return cmd
}
