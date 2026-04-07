package board

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newIssuesCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		jql        string
		maxResults int
		format     string
		fields     string
		raw        bool
	)

	cmd := &cobra.Command{
		Use:   "issues <board-id>",
		Short: "List issues on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid board ID: %s", args[0])
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.FetchBoardIssues(boardID, 0, maxResults, jql)
			if err != nil {
				return err
			}

			if format == "json" || raw {
				return output.JSON(data)
			}

			issuesRaw, ok := data["issues"]
			if !ok {
				fmt.Println("No issues found.")
				return nil
			}

			issues, ok := issuesRaw.([]any)
			if !ok {
				return output.JSON(data)
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

			if format == "ndjson" {
				return output.NDJSON(rows)
			}

			output.Table(rows, cols, "Board Issues")
			return nil
		},
	}

	cmd.Flags().StringVar(&jql, "jql", "", "JQL filter")
	cmd.Flags().IntVar(&maxResults, "max", 50, "Max results")
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json, ndjson)")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated columns")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
