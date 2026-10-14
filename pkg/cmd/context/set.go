package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newSetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		project   string
		boardID   int
		epic      string
		labels    []string
		issueType string
		status    string
		assignee  string
		display   string
	)

	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: "Create or update a named context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			name := args[0]
			ctx := config.GetContext(cfg, name)
			if ctx == nil {
				ctx = &config.Context{}
			}

			profileChanged := cmd.Flags().Changed("profile")
			if ctx.Profile == "" || profileChanged {
				ctx.Profile = f.ResolveProfileName(cfg)
			}
			if config.GetProfile(cfg, ctx.Profile) == nil {
				return fmt.Errorf("profile '%s' not found", ctx.Profile)
			}

			if cmd.Flags().Changed("project") {
				ctx.Project = project
			}
			if cmd.Flags().Changed("board-id") {
				ctx.BoardID = boardID
			}
			if cmd.Flags().Changed("epic") {
				ctx.Epic = epic
			}
			if cmd.Flags().Changed("label") {
				ctx.Labels = labels
			}
			if cmd.Flags().Changed("issue-type") {
				ctx.IssueType = issueType
			}
			if cmd.Flags().Changed("status") {
				ctx.Status = status
			}
			if cmd.Flags().Changed("assignee") {
				ctx.Assignee = assignee
			}
			if cmd.Flags().Changed("display") {
				if _, err := output.ParseFormat(display); err != nil {
					return err
				}
				ctx.Display = display
			}

			config.UpsertContext(cfg, name, ctx)
			if err := config.Save(cfg); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Context '%s' updated.", name)
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project key")
	cmd.Flags().IntVar(&boardID, "board-id", 0, "Board ID")
	cmd.Flags().StringVar(&epic, "epic", "", "Epic issue key")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Label (repeatable)")
	cmd.Flags().StringVar(&issueType, "issue-type", "", "Issue type")
	cmd.Flags().StringVar(&status, "status", "", "Status")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Assignee")
	cmd.Flags().StringVar(&display, "display", "", "Default output format: json, markdown")

	return cmd
}
