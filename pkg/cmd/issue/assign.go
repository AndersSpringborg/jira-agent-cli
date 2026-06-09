package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

type userSearcher interface {
	ListUsers(query string) ([]map[string]any, error)
}

func resolveAssignmentUser(client userSearcher, user string) (accountID, name string) {
	users, err := client.ListUsers(user)
	if err != nil || len(users) == 0 {
		return user, user
	}

	selected := users[0]
	for _, candidate := range users {
		if strings.EqualFold(stringField(candidate, "emailAddress"), user) || strings.EqualFold(stringField(candidate, "name"), user) {
			selected = candidate
			break
		}
	}

	accountID = stringField(selected, "accountId")
	name = stringField(selected, "name")
	if accountID == "" {
		accountID = user
	}
	if name == "" {
		name = user
	}
	return accountID, name
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

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
				return writeMutationResult(driver, map[string]any{
					"status": "unassigned",
					"key":    issueKey,
				})

			case "default":
				if err := client.AssignIssue(issueKey, "-1", "", ""); err != nil {
					return err
				}
				return writeMutationResult(driver, map[string]any{
					"status": "assigned_default",
					"key":    issueKey,
				})

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
				return writeMutationResult(driver, map[string]any{
					"status":       "assigned",
					"key":          issueKey,
					"user":         displayName,
					"accountId":    accountID,
					"current_user": true,
				})

			default:
				accountID, name := resolveAssignmentUser(client, user)
				if err := client.AssignIssue(issueKey, accountID, name, ""); err != nil {
					return err
				}
				return writeMutationResult(driver, map[string]any{
					"status": "assigned",
					"key":    issueKey,
					"user":   user,
				})
			}
		},
	}

	return cmd
}
