package context

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show active context for the current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := f.LoadProfile()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			ctx := profile.Context
			if ctx == nil || ctx.IsEmpty() {
				return driver.Message("No context set for profile '%s'.", profile.Name)
			}

			jql := config.BuildJQL(ctx)

			data := map[string]any{
				"profile": profile.Name,
			}
			if ctx.Project != "" {
				data["project"] = ctx.Project
			}
			if ctx.BoardID != 0 {
				data["board_id"] = ctx.BoardID
			}
			if ctx.Epic != "" {
				data["epic"] = ctx.Epic
			}
			if len(ctx.Labels) > 0 {
				data["labels"] = ctx.Labels
			}
			if ctx.IssueType != "" {
				data["issue_type"] = ctx.IssueType
			}
			if ctx.Status != "" {
				data["status"] = ctx.Status
			}
			if ctx.Assignee != "" {
				data["assignee"] = ctx.Assignee
			}
			if ctx.Display != "" {
				data["display"] = ctx.Display
			}
			if jql != "" {
				data["jql"] = jql
			}

			return driver.Item("Context", data)
		},
	}

	return cmd
}
