package auth

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newLogoutCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newWhoamiCmd(f))

	return cmd
}
