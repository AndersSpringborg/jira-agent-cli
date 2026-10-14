package auth

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/auth"
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check if a token is stored for the current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}
			profileName := f.ResolveProfileName(cfg)

			token, err := auth.GetToken(profileName)
			if err != nil {
				return fmt.Errorf("get token: %w", err)
			}

			driver := f.DisplayDriver(cmd)
			if token != "" {
				return driver.Message("Token available for profile '%s'.", profileName)
			}
			return driver.Message("No token for profile '%s'.", profileName)
		},
	}
}
