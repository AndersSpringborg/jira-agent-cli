package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show a named context (active context by default)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			name := cfg.ActiveContext
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no active context; create one with `jira context set <name>`")
			}
			ctx := config.GetContext(cfg, name)
			if ctx == nil {
				return fmt.Errorf("context '%s' not found", name)
			}

			data := map[string]any{
				"name":    name,
				"active":  name == cfg.ActiveContext,
				"profile": ctx.Profile,
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
			if jql := config.BuildJQL(ctx); jql != "" {
				data["jql"] = jql
			}

			driver := f.DisplayDriver(cmd)
			return driver.Item("Context", data)
		},
	}
}
