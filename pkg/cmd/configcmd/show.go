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
			profileName := f.ResolveProfileName(cfg)
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

			if profile.Context != nil && !profile.Context.IsEmpty() {
				ctx := map[string]any{}
				if profile.Context.Project != "" {
					ctx["project"] = profile.Context.Project
				}
				if profile.Context.BoardID != 0 {
					ctx["board_id"] = profile.Context.BoardID
				}
				if profile.Context.Epic != "" {
					ctx["epic"] = profile.Context.Epic
				}
				if len(profile.Context.Labels) > 0 {
					ctx["labels"] = profile.Context.Labels
				}
				if profile.Context.IssueType != "" {
					ctx["issue_type"] = profile.Context.IssueType
				}
				if profile.Context.Status != "" {
					ctx["status"] = profile.Context.Status
				}
				if profile.Context.Assignee != "" {
					ctx["assignee"] = profile.Context.Assignee
				}
				data["context"] = ctx
			}

			driver := f.DisplayDriver(cmd)
			return driver.Item("Profile", data)
		},
	}
}
