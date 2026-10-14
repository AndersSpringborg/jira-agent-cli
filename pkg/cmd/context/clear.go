package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"

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
		Use:   "clear [name]",
		Short: "Delete a named context or clear selected fields",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, contexts, err := openForChange(f, cmd, args)
			if err != nil {
				return err
			}

			name := contexts.ActiveContext
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no active context")
			}
			ctx := contexts.Get(name)
			if ctx == nil {
				return fmt.Errorf("context '%s' not found", name)
			}

			noSpecific := !project && !boardID && !epic && !labels && !issueType && !status && !assignee && !display
			if noSpecific {
				contexts.Delete(name)
				if contexts.ActiveContext == name {
					contexts.ActiveContext = ""
				}
			} else {
				if project {
					ctx.Project = ""
				}
				if boardID {
					ctx.BoardID = 0
				}
				if epic {
					ctx.Epic = ""
				}
				if labels {
					ctx.Labels = nil
				}
				if issueType {
					ctx.IssueType = ""
				}
				if status {
					ctx.Status = ""
				}
				if assignee {
					ctx.Assignee = ""
				}
				if display {
					ctx.Display = ""
				}
			}

			if err := store.Save(contexts); err != nil {
				return err
			}

			driver := f.DisplayDriverTo(cmd, cmd.OutOrStdout())
			if noSpecific {
				return driver.Message("Context '%s' deleted.", name)
			}
			return driver.Message("Context '%s' updated.", name)
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
