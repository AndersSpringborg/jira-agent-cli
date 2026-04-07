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
		epic       string
		labels     []string
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
Without flags, uses the filters from the active context.

Examples:
  jira issue list
  jira issue list --project PROJ --status "In Progress"
  jira issue list --assignee currentUser() --max 20
  jira issue list --epic PROJ-42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := f.LoadProfile()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			ctx := profile.Context

			// Merge flags over context: flags take priority.
			if project == "" && ctx != nil && ctx.Project != "" {
				project = ctx.Project
			}
			if project == "" {
				return fmt.Errorf("no project specified; use --project or set context with `jira context set --project PROJ`")
			}
			if assignee == "" && ctx != nil && ctx.Assignee != "" {
				assignee = ctx.Assignee
			}
			if status == "" && ctx != nil && ctx.Status != "" {
				status = ctx.Status
			}
			if issueType == "" && ctx != nil && ctx.IssueType != "" {
				issueType = ctx.IssueType
			}
			if epic == "" && ctx != nil && ctx.Epic != "" {
				epic = ctx.Epic
			}
			if len(labels) == 0 && ctx != nil && len(ctx.Labels) > 0 {
				labels = ctx.Labels
			}

			// Build JQL
			jqlParts := []string{fmt.Sprintf("project = %s", project)}
			if epic != "" {
				jqlParts = append(jqlParts, fmt.Sprintf(`"Epic Link" = %s`, epic))
			}
			if assignee != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("assignee = %s", assignee))
			}
			if status != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("status = \"%s\"", status))
			}
			if issueType != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("issuetype = \"%s\"", issueType))
			}
			if len(labels) > 0 {
				quoted := make([]string, len(labels))
				for i, l := range labels {
					quoted[i] = fmt.Sprintf(`"%s"`, l)
				}
				jqlParts = append(jqlParts, fmt.Sprintf("labels in (%s)", strings.Join(quoted, ", ")))
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

			return driver.List("Issues", cols, rows)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Filter by assignee (use 'currentUser()' for self)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "Filter by issue type")
	cmd.Flags().StringVar(&epic, "epic", "", "Filter by epic issue key")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Filter by label (repeatable)")
	cmd.Flags().IntVar(&maxResults, "max", 20, "Max results")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}
