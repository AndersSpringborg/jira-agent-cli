package context

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage active filters and defaults",
	}

	cmd.AddCommand(newSetCmd(f))
	cmd.AddCommand(newShowCmd(f))
	cmd.AddCommand(newClearCmd(f))

	return cmd
}
