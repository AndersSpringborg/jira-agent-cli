package search

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newJQLCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		maxResults int
		columns    string
		raw        bool
	)

	cmd := &cobra.Command{
		Use:   "jql <query>",
		Short: "Search issues with JQL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jql := args[0]

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.Search(jql, 0, maxResults)
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

			cols := output.NormalizeFields(columns, []string{"key", "summary", "status", "assignee", "priority"})
			rows := make([]map[string]any, 0, len(issues))
			for _, item := range issues {
				issue, ok := item.(map[string]any)
				if !ok {
					continue
				}
				row := map[string]any{
					"key": issue["key"],
				}
				flds, _ := issue["fields"].(map[string]any)
				if flds != nil {
					row["summary"] = flds["summary"]
					row["status"] = flds["status"]
					row["assignee"] = flds["assignee"]
					row["priority"] = flds["priority"]
					row["issuetype"] = flds["issuetype"]
					row["reporter"] = flds["reporter"]
					row["resolution"] = flds["resolution"]
					row["created"] = flds["created"]
					row["updated"] = flds["updated"]
				}
				rows = append(rows, row)
			}

			return driver.List("Issues", cols, rows)
		},
	}

	cmd.Flags().IntVar(&maxResults, "max", 50, "Max results")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}
