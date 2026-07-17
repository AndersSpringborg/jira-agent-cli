// Package cmd provides the root command for the jira CLI.
package cmd

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/pkg/cmd/api"
	"AndersSpringborg/jira-cli/pkg/cmd/auth"
	"AndersSpringborg/jira-cli/pkg/cmd/board"
	"AndersSpringborg/jira-cli/pkg/cmd/configcmd"
	cmdcontext "AndersSpringborg/jira-cli/pkg/cmd/context"
	"AndersSpringborg/jira-cli/pkg/cmd/feedback"
	"AndersSpringborg/jira-cli/pkg/cmd/install"
	"AndersSpringborg/jira-cli/pkg/cmd/issue"
	"AndersSpringborg/jira-cli/pkg/cmd/me"
	"AndersSpringborg/jira-cli/pkg/cmd/mine"
	"AndersSpringborg/jira-cli/pkg/cmd/open"
	"AndersSpringborg/jira-cli/pkg/cmd/ping"
	"AndersSpringborg/jira-cli/pkg/cmd/project"
	"AndersSpringborg/jira-cli/pkg/cmd/search"
	"AndersSpringborg/jira-cli/pkg/cmd/sprint"
	"AndersSpringborg/jira-cli/pkg/cmd/update"
	"AndersSpringborg/jira-cli/pkg/cmd/user"

	"github.com/spf13/cobra"
)

// Execute runs root and, on failure, prints the error and the failing
// command's complete help text to stderr so automated callers can recover.
func Execute(root *cobra.Command) error {
	executed, err := root.ExecuteC()
	if err == nil {
		return nil
	}
	if executed == nil {
		executed = root
	}
	executed.PrintErrln(err)
	executed.PrintErrln()
	executed.SetOut(executed.ErrOrStderr())
	if helpErr := executed.Help(); helpErr != nil {
		return fmt.Errorf("%w (render help: %v)", err, helpErr)
	}
	return err
}

// NewRootCmd creates the root cobra command for the jira CLI.
func NewRootCmd(version, date string) *cobra.Command {
	f := &cmdutil.Factory{}

	cmd := &cobra.Command{
		Use:     "jira",
		Short:   "Jira CLI — kubectl for Jira, designed for AI agents",
		Version: version,
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

	cmd.SetVersionTemplate("jira version " + version + " (built " + date + ")\n")
	f.DebugWriter = cmd.ErrOrStderr

	// Global flags
	cmd.PersistentFlags().StringVar(&f.Profile, "profile", "", "Config profile to use (default: from config or 'default')")
	cmd.PersistentFlags().String("format", "json", "Output format: json, markdown")
	cmd.PersistentFlags().BoolVar(&f.Debug, "debug", false, "Print raw Jira HTTP responses to stderr")

	// Register command groups
	cmd.AddCommand(api.NewCmd(f))
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
	cmd.AddCommand(mine.NewCmd(f))
	cmd.AddCommand(open.NewCmd(f))
	cmd.AddCommand(ping.NewCmd(f))
	cmd.AddCommand(update.NewCmd(version))
	cmd.AddCommand(install.NewCmd(cmd))
	cmd.AddCommand(feedback.NewCmd(f))

	return cmd
}
