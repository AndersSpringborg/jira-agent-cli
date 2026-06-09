package issue

import (
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <issue-key>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			if err := client.DeleteIssue(issueKey); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return writeMutationResult(driver, map[string]any{
				"status": "deleted",
				"key":    issueKey,
			})
		},
	}

	return cmd
}
