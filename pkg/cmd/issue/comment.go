package issue

import (
	"fmt"
	"os"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newCommentAddCmd(f *cmdutil.Factory) *cobra.Command {
	var template string

	cmd := &cobra.Command{
		Use:   "add <issue-key> [body]",
		Short: "Add a comment to an issue (body is Markdown, converted to Jira ADF/wiki markup)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			body, err := resolveCommentBody(args[1:], template)
			if err != nil {
				return err
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			if err := client.AddComment(issueKey, body); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return writeMutationResult(driver, map[string]any{
				"status": "comment_added",
				"key":    issueKey,
			})
		},
	}

	cmd.Flags().StringVar(&template, "template", "", "Load comment body from template file")

	return cmd
}

func resolveCommentBody(args []string, template string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	if template == "" {
		return "", fmt.Errorf("comment body is required (pass as argument or use --template)")
	}
	data, err := os.ReadFile(template)
	if err != nil {
		return "", fmt.Errorf("read comment template: %w", err)
	}
	body := strings.TrimSpace(string(data))
	if body == "" {
		return "", fmt.Errorf("comment template %q is empty", template)
	}
	return body, nil
}

func newCommentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}

	cmd.AddCommand(newCommentAddCmd(f))

	return cmd
}
