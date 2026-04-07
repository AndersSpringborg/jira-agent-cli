package configcmd

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func NewInitCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		profile        string
		baseURL        string
		timeoutSeconds float64
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			_ = f

			p := &config.Profile{
				Name:           profile,
				BaseURL:        baseURL,
				TimeoutSeconds: timeoutSeconds,
			}
			config.UpsertProfile(cfg, p)
			cfg.DefaultProfile = profile
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Profile '%s' saved and set as default.\n", profile)
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", config.DefaultProfile, "Profile name")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Jira base URL")
	cmd.Flags().Float64Var(&timeoutSeconds, "timeout", 15.0, "HTTP timeout in seconds")
	_ = cmd.MarkFlagRequired("base-url")

	return cmd
}
