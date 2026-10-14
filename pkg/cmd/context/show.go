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
			store, err := f.ContextStore()
			if err != nil {
				return err
			}
			contexts, err := store.Load()
			if err != nil {
				return err
			}

			name := contexts.ActiveContext
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no active context; ask the user to create one with `sudo jira context set <name>`")
			}
			ctx := contexts.Get(name)
			if ctx == nil {
				return fmt.Errorf("context '%s' not found", name)
			}

			data := map[string]any{
				"name":      name,
				"active":    name == contexts.ActiveContext,
				"profile":   ctx.Profile,
				"path":      store.Path(),
				"protected": true,
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

			driver := f.DisplayDriverTo(cmd, cmd.OutOrStdout())
			return driver.Item("Context", data)
		},
	}
}
