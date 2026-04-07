package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newMoveCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		comment    string
		resolution string
		assignee   string
	)

	cmd := &cobra.Command{
		Use:     "move <issue-key> [state]",
		Aliases: []string{"transition"},
		Short:   "Transition an issue to a given state",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			transitions, err := client.GetIssueTransitions(issueKey)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if len(args) < 2 {
				rows := make([]map[string]any, 0, len(transitions))
				for _, t := range transitions {
					rows = append(rows, map[string]any{
						"name": t["name"],
						"id":   t["id"],
					})
				}
				return driver.List("Transitions", []string{"name", "id"}, rows)
			}

			targetState := args[1]
			var transitionID string
			targetLower := strings.ToLower(targetState)

			for _, t := range transitions {
				name, _ := t["name"].(string)
				if strings.EqualFold(name, targetState) {
					transitionID, _ = t["id"].(string)
					break
				}
				if strings.Contains(strings.ToLower(name), targetLower) {
					transitionID, _ = t["id"].(string)
				}
			}

			if transitionID == "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "No transition found matching '%s'. Available:\n", targetState)
				for _, t := range transitions {
					fmt.Fprintf(cmd.ErrOrStderr(), "  %s (id: %v)\n", t["name"], t["id"])
				}
				return fmt.Errorf("invalid transition: %s", targetState)
			}

			if err := client.TransitionIssue(issueKey, transitionID); err != nil {
				return err
			}

			_ = comment
			_ = resolution
			_ = assignee

			return driver.Message("Transitioned %s to %s", issueKey, targetState)
		},
	}

	cmd.Flags().StringVar(&comment, "comment", "", "Add a comment during transition")
	cmd.Flags().StringVarP(&resolution, "resolution", "R", "", "Set resolution during transition")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Set assignee during transition")

	return cmd
}
