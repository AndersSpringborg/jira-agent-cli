package configcmd

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show profile details",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}
			profileName, err := f.ResolveProfileName(cfg)
			if err != nil {
				return err
			}
			profile := config.GetProfile(cfg, profileName)
			if profile == nil {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			data := map[string]any{
				"name":            profile.Name,
				"base_url":        profile.BaseURL,
				"user_email":      profile.UserEmail,
				"timeout_seconds": profile.TimeoutSeconds,
			}

			driver := f.DisplayDriver(cmd)
			return driver.Item("Profile", data)
		},
	}
}
