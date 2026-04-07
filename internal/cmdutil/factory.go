package cmdutil

import (
	"fmt"
	"os"

	"AndersSpringborg/jira-cli/internal/auth"
	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/jira"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

type Factory struct {
	Profile string
}

func (f *Factory) LoadConfig() (*config.Config, error) {
	return config.Load()
}

func (f *Factory) ResolveProfileName(cfg *config.Config) string {
	return config.ResolveProfileName(cfg, f.Profile)
}

func (f *Factory) LoadProfile() (*config.Profile, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	profileName := config.ResolveProfileName(cfg, f.Profile)
	profile := config.GetProfile(cfg, profileName)
	if profile == nil {
		return &config.Profile{Name: profileName}, nil
	}
	return profile, nil
}

// DisplayDriver resolves the output format and returns the appropriate driver.
//
// Resolution order:
//  1. --format flag (if explicitly set on the command line)
//  2. context.Display (persisted via `jira context set --display markdown`)
//  3. "json" (hardcoded default)
func (f *Factory) DisplayDriver(cmd *cobra.Command) output.DisplayDriver {
	// Check if --format was explicitly passed on the command line.
	if cmd != nil {
		if flag := cmd.Root().PersistentFlags().Lookup("format"); flag != nil && flag.Changed {
			if format, err := output.ParseFormat(flag.Value.String()); err == nil {
				return output.NewDriver(format)
			}
		}
	}

	// Fall back to context display setting.
	profile, err := f.LoadProfile()
	if err == nil && profile.Context != nil && profile.Context.Display != "" {
		if format, err := output.ParseFormat(profile.Context.Display); err == nil {
			return output.NewDriver(format)
		}
	}

	// Default to JSON.
	return output.NewDriver(output.FormatJSON)
}

func (f *Factory) LoadClient() (*jira.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	profileName := config.ResolveProfileName(cfg, f.Profile)
	profile := config.GetProfile(cfg, profileName)

	baseURL := os.Getenv("JIRA_BASE_URL")
	if baseURL == "" && profile != nil {
		baseURL = profile.BaseURL
	}
	if baseURL == "" {
		return nil, notLoggedInError(profileName, "no server URL configured")
	}

	token := os.Getenv("JIRA_TOKEN")
	if token == "" {
		var err error
		token, err = auth.GetToken(profileName)
		if err != nil {
			return nil, notLoggedInError(profileName, fmt.Sprintf("could not read token from keychain: %v", err))
		}
	}
	if token == "" {
		return nil, notLoggedInError(profileName, "no API token found")
	}

	email := os.Getenv("JIRA_EMAIL")
	if email == "" && profile != nil {
		email = profile.UserEmail
	}

	authType := os.Getenv("JIRA_AUTH_TYPE")
	if authType == "" && profile != nil {
		authType = profile.AuthType
	}
	if authType == "" {
		authType = config.DetectAuthType(baseURL)
	}

	timeout := 15.0
	if profile != nil && profile.TimeoutSeconds > 0 {
		timeout = profile.TimeoutSeconds
	}

	return jira.NewClient(baseURL, email, token, authType, timeout)
}

func notLoggedInError(profile, reason string) error {
	return fmt.Errorf(`not logged in (%s)

To authenticate, run:

  Jira Cloud:
    jira auth login --server https://YOUR-ORG.atlassian.net --email YOU@EXAMPLE.COM --token YOUR_API_TOKEN

  Jira Server / Data Center (PAT):
    jira auth login --server https://jira.example.com --token YOUR_PAT

  Or set environment variables:
    export JIRA_BASE_URL=https://YOUR-ORG.atlassian.net
    export JIRA_TOKEN=YOUR_API_TOKEN
    export JIRA_EMAIL=YOU@EXAMPLE.COM

Profile: %s`, reason, profile)
}
