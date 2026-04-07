package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newClearCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		project   bool
		boardID   bool
		epic      bool
		labels    bool
		issueType bool
		status    bool
		assignee  bool
		display   bool
	)

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear context filters (all or specific)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			profileName := f.ResolveProfileName(cfg)
			profile := config.GetProfile(cfg, profileName)
			if profile == nil || profile.Context == nil {
				fmt.Printf("No context to clear for profile '%s'.\n", profileName)
				return nil
			}

			noSpecific := !project && !boardID && !epic && !labels && !issueType && !status && !assignee && !display

			if noSpecific {
				profile.Context = nil
			} else {
				if project {
					profile.Context.Project = ""
				}
				if boardID {
					profile.Context.BoardID = 0
				}
				if epic {
					profile.Context.Epic = ""
				}
				if labels {
					profile.Context.Labels = nil
				}
				if issueType {
					profile.Context.IssueType = ""
				}
				if status {
					profile.Context.Status = ""
				}
				if assignee {
					profile.Context.Assignee = ""
				}
				if display {
					profile.Context.Display = ""
				}
				if profile.Context.IsEmpty() {
					profile.Context = nil
				}
			}

			config.UpsertProfile(cfg, profile)
			if err := config.Save(cfg); err != nil {
				return err
			}

			if noSpecific {
				fmt.Printf("Context cleared for profile '%s'.\n", profileName)
			} else {
				fmt.Printf("Context updated for profile '%s'.\n", profileName)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&project, "project", false, "Clear project filter")
	cmd.Flags().BoolVar(&boardID, "board-id", false, "Clear board ID")
	cmd.Flags().BoolVar(&epic, "epic", false, "Clear epic filter")
	cmd.Flags().BoolVar(&labels, "label", false, "Clear labels filter")
	cmd.Flags().BoolVar(&issueType, "issue-type", false, "Clear issue type filter")
	cmd.Flags().BoolVar(&status, "status", false, "Clear status filter")
	cmd.Flags().BoolVar(&assignee, "assignee", false, "Clear assignee filter")
	cmd.Flags().BoolVar(&display, "display", false, "Clear display format")

	return cmd
}
