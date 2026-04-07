package issue

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	listCmd := newListCmd(f)

	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
		// Default to `issue list` when no subcommand is given.
		RunE: listCmd.RunE,
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newEditCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newAssignCmd(f))
	cmd.AddCommand(newMoveCmd(f))
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newBrowseCmd(f))
	cmd.AddCommand(newLinkCmd(f))
	cmd.AddCommand(newUnlinkCmd(f))
	cmd.AddCommand(newCloneCmd(f))

	return cmd
}
