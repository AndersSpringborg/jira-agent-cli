package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

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
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set context filters for the active profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			profileName := f.ResolveProfileName(cfg)
			profile := config.GetProfile(cfg, profileName)
			if profile == nil {
				profile = &config.Profile{Name: profileName}
			}

			if profile.Context == nil {
				profile.Context = &config.Context{}
			}

			if cmd.Flags().Changed("project") {
				profile.Context.Project = project
			}
			if cmd.Flags().Changed("board-id") {
				profile.Context.BoardID = boardID
			}
			if cmd.Flags().Changed("epic") {
				profile.Context.Epic = epic
			}
			if cmd.Flags().Changed("label") {
				profile.Context.Labels = labels
			}
			if cmd.Flags().Changed("issue-type") {
				profile.Context.IssueType = issueType
			}
			if cmd.Flags().Changed("status") {
				profile.Context.Status = status
			}
			if cmd.Flags().Changed("assignee") {
				profile.Context.Assignee = assignee
			}

			config.UpsertProfile(cfg, profile)
			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Printf("Context updated for profile '%s'.\n", profileName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project key")
	cmd.Flags().IntVar(&boardID, "board-id", 0, "Board ID")
	cmd.Flags().StringVar(&epic, "epic", "", "Epic issue key")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Label (repeatable)")
	cmd.Flags().StringVar(&issueType, "issue-type", "", "Issue type")
	cmd.Flags().StringVar(&status, "status", "", "Status")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Assignee")

	return cmd
}
