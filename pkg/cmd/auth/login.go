package auth

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/auth"
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		token    string
		email    string
		baseURL  string
		authType string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store authentication credentials",
		Long: `Store your Jira API token in the OS keychain.

Jira Cloud (*.atlassian.net):
  1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
  2. Click "Create API token", give it a label, and copy the token.
  3. Run:
       jira auth login --server https://your-org.atlassian.net \
         --email you@example.com --token YOUR_API_TOKEN

Jira Server / Data Center (Personal Access Token):
  1. In Jira, go to Profile -> Personal Access Tokens
     (usually https://jira.example.com/secure/ViewProfile.jspa -> Personal Access Tokens)
  2. Click "Create token", give it a label, and copy the token.
  3. Run:
       jira auth login --server https://jira.example.com --token YOUR_PAT`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				return fmt.Errorf("--token is required")
			}

			cfg, err := config.Load()
			if err != nil {
				// Config doesn't exist yet, create it
				cfg = &config.Config{
					Profiles: make(map[string]*config.Profile),
				}
			}

			profileName := config.ResolveProfileName(cfg, f.Profile)
			profile := config.GetProfile(cfg, profileName)
			if profile == nil {
				profile = &config.Profile{Name: profileName}
			}

			if baseURL != "" {
				profile.BaseURL = baseURL
			}
			if email != "" {
				profile.UserEmail = email
			}
			if authType != "" {
				profile.AuthType = authType
			} else if profile.AuthType == "" {
				profile.AuthType = config.DetectAuthType(profile.BaseURL)
			}

			if profile.BaseURL == "" {
				return fmt.Errorf("--server is required (or set base_url in config)")
			}

			// Store token in keychain
			if err := auth.SetToken(profileName, token); err != nil {
				return fmt.Errorf("store token: %w", err)
			}

			// Save profile to config
			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]*config.Profile)
			}
			cfg.Profiles[profileName] = profile
			if cfg.DefaultProfile == "" {
				cfg.DefaultProfile = profileName
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Token stored for profile '%s'", profileName)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "API token or PAT (required)")
	cmd.Flags().StringVar(&email, "email", "", "User email (required for Jira Cloud)")
	cmd.Flags().StringVar(&baseURL, "server", "", "Jira server URL")
	cmd.Flags().StringVar(&authType, "auth-type", "", "Auth type: basic, pat")

	return cmd
}
