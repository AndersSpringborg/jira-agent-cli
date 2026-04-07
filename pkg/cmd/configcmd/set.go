package configcmd

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newSetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		profile        string
		baseURL        string
		userEmail      string
		timeoutSeconds float64
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			existing := config.GetProfile(cfg, profile)
			if existing == nil {
				existing = &config.Profile{Name: profile}
			}

			if cmd.Flags().Changed("base-url") {
				existing.BaseURL = baseURL
			}
			if cmd.Flags().Changed("user-email") {
				existing.UserEmail = userEmail
			}
			if cmd.Flags().Changed("timeout") {
				existing.TimeoutSeconds = timeoutSeconds
			}

			config.UpsertProfile(cfg, existing)
			if err := config.Save(cfg); err != nil {
				return err
			}
			driver := f.DisplayDriver(cmd)
			return driver.Message("Profile '%s' updated.", profile)
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile name")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Jira base URL")
	cmd.Flags().StringVar(&userEmail, "user-email", "", "User email")
	cmd.Flags().Float64Var(&timeoutSeconds, "timeout", 15.0, "HTTP timeout in seconds")
	_ = cmd.MarkFlagRequired("profile")

	return cmd
}
