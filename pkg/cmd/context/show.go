package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format string
		raw    bool
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show active context for the current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := f.LoadProfile()
			if err != nil {
				return err
			}

			ctx := profile.Context
			if ctx == nil || ctx.IsEmpty() {
				fmt.Printf("No context set for profile '%s'.\n", profile.Name)
				return nil
			}

			jql := config.BuildJQL(ctx)

			data := map[string]any{}
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
			if jql != "" {
				data["jql"] = jql
			}

			if format == "json" || raw {
				return output.JSON(data)
			}

			fmt.Printf("Context for profile '%s':\n", profile.Name)
			for k, v := range data {
				fmt.Printf("  %-12s %v\n", k+":", v)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
