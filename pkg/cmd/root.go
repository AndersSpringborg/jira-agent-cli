// Package cmd provides the root command for the jira CLI.
package cmd

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/pkg/cmd/auth"
	"AndersSpringborg/jira-cli/pkg/cmd/board"
	"AndersSpringborg/jira-cli/pkg/cmd/configcmd"
	cmdcontext "AndersSpringborg/jira-cli/pkg/cmd/context"
	"AndersSpringborg/jira-cli/pkg/cmd/install"
	"AndersSpringborg/jira-cli/pkg/cmd/issue"
	"AndersSpringborg/jira-cli/pkg/cmd/me"
	"AndersSpringborg/jira-cli/pkg/cmd/open"
	"AndersSpringborg/jira-cli/pkg/cmd/ping"
	"AndersSpringborg/jira-cli/pkg/cmd/project"
	"AndersSpringborg/jira-cli/pkg/cmd/search"
	"AndersSpringborg/jira-cli/pkg/cmd/sprint"
	"AndersSpringborg/jira-cli/pkg/cmd/user"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command for the jira CLI.
func NewRootCmd() *cobra.Command {
	f := &cmdutil.Factory{}

	cmd := &cobra.Command{
		Use:   "jira",
		Short: "Jira CLI — kubectl for Jira, designed for AI agents",
		Long: `A non-interactive CLI for Jira designed for AI agents and automation.

Output formats:
  --format json       Machine-readable JSON (default). Pipe to jq.
  --format markdown   Structured markdown for LLM consumption.

Examples:
  jira issue view PROJ-123
  jira issue view PROJ-123 --format markdown
  jira search jql "project = PROJ AND status = 'In Progress'" | jq '.[].key'
  jira issue create -p PROJ -s "Fix login bug" -t Bug
  jira sprint list 42 --state active`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	cmd.PersistentFlags().StringVar(&f.Profile, "profile", "", "Config profile to use (default: from config or 'default')")
	cmd.PersistentFlags().String("format", "json", "Output format: json, markdown")

	// Register command groups
	cmd.AddCommand(auth.NewCmd(f))
	cmd.AddCommand(configcmd.NewCmd(f))
	cmd.AddCommand(cmdcontext.NewCmd(f))
	cmd.AddCommand(issue.NewCmd(f))
	cmd.AddCommand(board.NewCmd(f))
	cmd.AddCommand(sprint.NewCmd(f))
	cmd.AddCommand(project.NewCmd(f))
	cmd.AddCommand(search.NewCmd(f))
	cmd.AddCommand(user.NewCmd(f))
	cmd.AddCommand(me.NewCmd(f))
	cmd.AddCommand(open.NewCmd(f))
	cmd.AddCommand(ping.NewCmd(f))
	cmd.AddCommand(install.NewCmd(cmd))

	return cmd
}
