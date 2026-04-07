package search

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search issues",
	}

	cmd.AddCommand(newJQLCmd(f))
	cmd.AddCommand(newTextCmd(f))

	return cmd
}
