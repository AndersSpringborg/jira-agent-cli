// Package boundary implements `jira boundary`, the sudo-protected limit on
// which issues the CLI may see or change.
package boundary

import (
	"errors"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boundary",
		Short: "Show or (with sudo) change which issues this CLI may access",
		Long: `The boundary limits every Jira request to issues in the given projects that
carry every given label. Anyone can read it; changing it requires sudo
(Administrator on Windows), so AI agents cannot widen their own access.

The context ("jira context") is different: it is the agent's working focus,
changes without sudo, and is always applied inside the boundary.

Examples:
  jira boundary show
  sudo jira boundary set --project PROJ --label safe
  sudo jira boundary clear`,
	}

	cmd.AddCommand(newShowCmd(f))
	cmd.AddCommand(newSetCmd(f))
	cmd.AddCommand(newClearCmd(f))
	return cmd
}

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the active boundary (no sudo needed)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := f.BoundaryStore()
			if err != nil {
				return err
			}
			boundary, err := store.Load()
			if err != nil {
				return err
			}
			data := map[string]any{
				"active":    boundary.Active(),
				"path":      store.Path(),
				"protected": true,
			}
			if boundary.Active() {
				data["projects"] = boundary.Projects
				data["labels"] = boundary.Labels
				data["jql"] = boundary.JQL()
			}
			return f.DisplayDriverTo(cmd, cmd.OutOrStdout()).Item("Boundary", data)
		},
	}
}

func newSetCmd(f *cmdutil.Factory) *cobra.Command {
	var projects, labels []string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Replace the boundary (requires sudo)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openForChange(f, cmd, args)
			if err != nil {
				return err
			}
			boundary := &config.Boundary{Projects: projects, Labels: labels}
			if !boundary.Active() {
				return errors.New("a boundary needs at least one --project or --label; use `jira boundary clear` to remove it")
			}
			if err := store.Save(boundary); err != nil {
				return err
			}
			return f.DisplayDriverTo(cmd, cmd.OutOrStdout()).Message("Boundary set: %s", boundary.JQL())
		},
	}

	cmd.Flags().StringSliceVar(&projects, "project", nil, "Allowed project key (repeatable; issues may be in any of them)")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Required label (repeatable; issues must carry every one)")
	return cmd
}

func newClearCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove the boundary (requires sudo)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openForChange(f, cmd, args)
			if err != nil {
				return err
			}
			if err := store.Clear(); err != nil {
				return err
			}
			return f.DisplayDriverTo(cmd, cmd.OutOrStdout()).Message("Boundary cleared.")
		},
	}
}
