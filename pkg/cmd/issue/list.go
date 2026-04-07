package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		project    string
		assignee   string
		status     string
		issueType  string
		maxResults int
		columns    string
		raw        bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List issues using context filters",
		Long: `List issues for the current project context.

Builds a JQL query from the provided flags and context settings.
Without flags, uses the project from the active context.

Examples:
  jira issue list
  jira issue list --project PROJ --status "In Progress"
  jira issue list --assignee currentUser() --max 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := f.LoadProfile()
			if err != nil {
				return err
			}

			// Build JQL from flags and context
			if project == "" && profile.Context != nil && profile.Context.Project != "" {
				project = profile.Context.Project
			}
			if project == "" {
				return fmt.Errorf("no project specified; use --project or set context with `jira context set --project PROJ`")
			}

			jqlParts := []string{fmt.Sprintf("project = %s", project)}
			if assignee != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("assignee = %s", assignee))
			} else if profile.Context != nil && profile.Context.Assignee != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("assignee = %s", profile.Context.Assignee))
			}
			if status != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("status = \"%s\"", status))
			} else if profile.Context != nil && profile.Context.Status != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("status = \"%s\"", profile.Context.Status))
			}
			if issueType != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("issuetype = \"%s\"", issueType))
			} else if profile.Context != nil && profile.Context.IssueType != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("issuetype = \"%s\"", profile.Context.IssueType))
			}

			jql := strings.Join(jqlParts, " AND ") + " ORDER BY updated DESC"

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.Search(jql, 0, maxResults)
			if err != nil {
				return err
			}

			if raw {
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

			cols := output.NormalizeFields(columns, []string{"key", "summary", "status", "assignee", "priority"})
			rows := make([]map[string]any, 0, len(issues))
			for _, item := range issues {
				iss, ok := item.(map[string]any)
				if !ok {
					continue
				}
				row := map[string]any{
					"key": iss["key"],
				}
				flds, _ := iss["fields"].(map[string]any)
				if flds != nil {
					row["summary"] = flds["summary"]
					row["status"] = flds["status"]
					row["assignee"] = flds["assignee"]
					row["priority"] = flds["priority"]
					row["issuetype"] = flds["issuetype"]
					row["reporter"] = flds["reporter"]
					row["created"] = flds["created"]
					row["updated"] = flds["updated"]
				}
				rows = append(rows, row)
			}

			output.TableWithOptions(rows, cols, "Issues", output.TableOptions{})
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Filter by assignee (use 'currentUser()' for self)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "Filter by issue type")
	cmd.Flags().IntVar(&maxResults, "max", 20, "Max results")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}
