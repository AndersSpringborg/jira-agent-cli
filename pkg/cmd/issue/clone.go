package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newCloneCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		summary   string
		priority  string
		assignee  string
		labels    []string
		replace   string
		rawOutput bool
	)

	cmd := &cobra.Command{
		Use:   "clone <issue-key>",
		Short: "Clone an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			overrides := map[string]any{}
			if summary != "" {
				overrides["summary"] = summary
			}
			if priority != "" {
				overrides["priority"] = map[string]any{"name": priority}
			}
			if assignee != "" {
				overrides["assignee"] = map[string]any{"name": assignee}
			}
			if len(labels) > 0 {
				overrides["labels"] = labels
			}

			data, err := client.CloneIssue(issueKey, overrides)
			if err != nil {
				return err
			}

			if replace != "" {
				_ = replace
			}

			if rawOutput {
				return output.JSON(data)
			}

			fmt.Printf("Cloned %s to %v\n", issueKey, data["key"])
			return nil
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Override summary")
	cmd.Flags().StringVarP(&priority, "priority", "y", "", "Override priority")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Override assignee")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Override labels")
	cmd.Flags().StringVarP(&replace, "replace", "H", "", "Replace string in summary/description (find:replace)")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Print raw JSON")

	return cmd
}
