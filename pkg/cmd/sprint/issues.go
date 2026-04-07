package sprint

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newIssuesCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		fields string
		raw    bool
	)

	cmd := &cobra.Command{
		Use:   "issues <sprint-id>",
		Short: "List issues in a sprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID: %s", args[0])
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetSprintIssues(sprintID)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(data)
			}

			issuesRaw, ok := data["issues"]
			if !ok {
				return driver.Message("No issues found.")
			}

			issues, ok := issuesRaw.([]any)
			if !ok {
				return driver.Raw(data)
			}

			cols := output.NormalizeFields(fields, []string{"key", "summary", "status", "assignee", "priority"})
			rows := make([]map[string]any, 0, len(issues))
			for _, item := range issues {
				issue, ok := item.(map[string]any)
				if !ok {
					continue
				}
				row := map[string]any{
					"key": issue["key"],
				}
				f, _ := issue["fields"].(map[string]any)
				if f != nil {
					row["summary"] = f["summary"]
					row["status"] = f["status"]
					row["assignee"] = f["assignee"]
					row["priority"] = f["priority"]
				}
				rows = append(rows, row)
			}

			return driver.List("Sprint Issues", cols, rows)
		},
	}

	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated columns")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
