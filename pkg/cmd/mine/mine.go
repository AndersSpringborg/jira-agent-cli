package mine

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"
	"AndersSpringborg/jira-cli/pkg/cmd/audit"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		project    string
		status     string
		issueType  string
		epic       string
		labels     []string
		maxResults int
		columns    string
		fields     []string
		raw        bool
		all        bool
	)

	cmd := &cobra.Command{
		Use:     "mine",
		Aliases: []string{"my"},
		Short:   "List issues assigned to you",
		Long: `List issues assigned to the current user across all projects.

Respects context settings (project, status, type, etc.) as defaults.
Excludes completed issues unless --all is specified.

Examples:
  jira mine
  jira mine --all
  jira mine --status "In Progress"
  jira mine --project PROJ --max 50`,
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
			jqlParts := []string{"assignee = currentUser()"}
			if project != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("project = %s", project))
			}
			if !all {
				jqlParts = append(jqlParts, "statusCategory != Done")
			}
			if epic != "" {
				jqlParts = append(jqlParts, fmt.Sprintf(`("Epic Link" = "%s" OR parent = "%s")`, epic, epic)) //nolint:gocritic // JQL requires double-quoted strings, not Go %q escaping
			}
			if status != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("status = \"%s\"", status)) //nolint:gocritic // JQL syntax
			}
			if issueType != "" {
				jqlParts = append(jqlParts, fmt.Sprintf("issuetype = \"%s\"", issueType)) //nolint:gocritic // JQL syntax
			}
			if len(labels) > 0 {
				quoted := make([]string, len(labels))
				for i, l := range labels {
					quoted[i] = fmt.Sprintf(`"%s"`, l) //nolint:gocritic // JQL syntax
				}
				jqlParts = append(jqlParts, fmt.Sprintf("labels in (%s)", strings.Join(quoted, ", ")))
			}

			jql := strings.Join(jqlParts, " AND ") + " ORDER BY updated DESC"

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.Search(jql, 0, maxResults, fields...)
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

			cols := output.NormalizeFields(columns, []string{"key", "summary", "status", "priority", "updated"})
			cols = output.AppendColumns(cols, fields)
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
					for _, fid := range fields {
						row[fid] = flds[fid]
					}
				}
				rows = append(rows, row)
			}

			return driver.List("My Issues", cols, rows)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "Filter by issue type")
	cmd.Flags().StringVar(&epic, "epic", "", "Filter by epic issue key")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Filter by label (repeatable)")
	cmd.Flags().IntVar(&maxResults, "max", 20, "Max results")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().StringArrayVarP(&fields, "field", "F", nil, "Custom field ID to fetch and display (e.g. customfield_10145), repeatable")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&all, "all", false, "Include completed issues")

	cmd.AddCommand(audit.NewCmd(f))

	return cmd
}
