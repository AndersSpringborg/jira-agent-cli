package sprint

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage sprints",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newGetCmd(f))
	cmd.AddCommand(newIssuesCmd(f))
	cmd.AddCommand(newStartCmd(f))
	cmd.AddCommand(newCloseCmd(f))
	cmd.AddCommand(newAddCmd(f))

	return cmd
}
