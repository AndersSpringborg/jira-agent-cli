package configcmd

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration profiles",
	}

	cmd.AddCommand(NewInitCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newShowCmd(f))
	cmd.AddCommand(newSetCmd(f))
	cmd.AddCommand(newUseCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}
