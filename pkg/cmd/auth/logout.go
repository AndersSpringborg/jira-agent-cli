package auth

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/auth"
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored token from the system keychain",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}
			profileName, err := f.ResolveProfileName(cfg)
			if err != nil {
				return err
			}

			deleted, err := auth.DeleteToken(profileName)
			if err != nil {
				return fmt.Errorf("delete token: %w", err)
			}

			driver := f.DisplayDriver(cmd)
			if deleted {
				return driver.Message("Removed token for profile '%s'.", profileName)
			}
			return driver.Message("No token found for profile '%s'.", profileName)
		},
	}
}
