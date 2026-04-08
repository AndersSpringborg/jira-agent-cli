package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newAssignCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign <issue-key> <user>",
		Short: "Assign an issue to a user",
		Long:  "Assign a user to an issue. Use 'me' to assign to yourself. Use 'x' to unassign. Use 'default' for default assignee.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			if len(args) < 2 {
				return fmt.Errorf("user is required (use 'me' for yourself, 'x' to unassign)")
			}

			user := args[1]

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			switch user {
			case "x":
				if err := client.AssignIssue(issueKey, "", "", ""); err != nil {
					return err
				}
				return driver.Message("Unassigned issue: %s", issueKey)

			case "default":
				if err := client.AssignIssue(issueKey, "-1", "", ""); err != nil {
					return err
				}
				return driver.Message("Assigned issue %s to default assignee", issueKey)

			case "me":
				data, err := client.GetMyself()
				if err != nil {
					return fmt.Errorf("failed to resolve current user: %w", err)
				}
				accountID, ok := data["accountId"].(string)
				if !ok || accountID == "" {
					return fmt.Errorf("could not determine account ID from current user")
				}
				if err := client.AssignIssue(issueKey, accountID, "", ""); err != nil {
					return err
				}
				displayName, _ := data["displayName"].(string)
				if displayName == "" {
					displayName = accountID
				}
				return driver.Message("Assigned issue %s to %s (me)", issueKey, displayName)

			default:
				if err := client.AssignIssue(issueKey, user, user, ""); err != nil {
					return err
				}
				return driver.Message("Assigned issue %s to %s", issueKey, user)
			}
		},
	}

	return cmd
}
