package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newCommentAddCmd(f *cmdutil.Factory) *cobra.Command {
	var template string

	cmd := &cobra.Command{
		Use:   "add <issue-key> [body]",
		Short: "Add a comment to an issue",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			var body string
			if len(args) > 1 {
				body = args[1]
			}

			if body == "" {
				return fmt.Errorf("comment body is required (pass as argument or use --template)")
			}

			_ = template

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			if err := client.AddComment(issueKey, body); err != nil {
				return err
			}

			fmt.Printf("Added comment to: %s\n", issueKey)
			return nil
		},
	}

	cmd.Flags().StringVar(&template, "template", "", "Load comment body from template file")

	return cmd
}

func newCommentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}

	cmd.AddCommand(newCommentAddCmd(f))

	return cmd
}
