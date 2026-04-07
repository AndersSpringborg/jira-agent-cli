package issue

import (
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newLinkCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <issue-1> <issue-2> <link-type>",
		Short: "Link two issues",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			issue1 := strings.ToUpper(args[0])
			issue2 := strings.ToUpper(args[1])
			linkType := args[2]

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			if err := client.LinkIssues(issue1, issue2, linkType); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Linked %s %s %s", issue1, linkType, issue2)
		},
	}

	return cmd
}
