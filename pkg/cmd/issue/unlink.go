package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newUnlinkCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlink <issue-1> <issue-2>",
		Short: "Unlink two linked issues",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issue1 := strings.ToUpper(args[0])
			issue2 := strings.ToUpper(args[1])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			links, err := client.GetIssueLinks(issue1)
			if err != nil {
				return err
			}

			for _, link := range links {
				inward, _ := link["inwardIssue"].(map[string]any)
				outward, _ := link["outwardIssue"].(map[string]any)
				inKey, _ := inward["key"].(string)
				outKey, _ := outward["key"].(string)

				if strings.EqualFold(inKey, issue2) || strings.EqualFold(outKey, issue2) {
					linkID := fmt.Sprintf("%v", link["id"])
					if err := client.DeleteIssueLink(linkID); err != nil {
						return err
					}
					fmt.Printf("Unlinked %s and %s\n", issue1, issue2)
					return nil
				}
			}

			return fmt.Errorf("no link found between %s and %s", issue1, issue2)
		},
	}

	return cmd
}
