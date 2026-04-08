package config

import (
	"fmt"
	"strings"
)

func BuildJQL(ctx *Context) string {
	if ctx == nil {
		return ""
	}

	var clauses []string

	if ctx.Project != "" {
		clauses = append(clauses, fmt.Sprintf(`project = "%s"`, ctx.Project)) //nolint:gocritic // JQL requires double-quoted strings, not Go %q escaping
	}
	if ctx.Epic != "" {
		clauses = append(clauses, fmt.Sprintf(`"Epic Link" = "%s"`, ctx.Epic)) //nolint:gocritic // JQL syntax
	}
	if len(ctx.Labels) > 0 {
		quoted := make([]string, len(ctx.Labels))
		for i, l := range ctx.Labels {
			quoted[i] = fmt.Sprintf(`"%s"`, l) //nolint:gocritic // JQL syntax
		}
		clauses = append(clauses, fmt.Sprintf("labels in (%s)", strings.Join(quoted, ", ")))
	}
	if ctx.IssueType != "" {
		clauses = append(clauses, fmt.Sprintf(`issuetype = "%s"`, ctx.IssueType)) //nolint:gocritic // JQL syntax
	}
	if ctx.Status != "" {
		clauses = append(clauses, fmt.Sprintf(`status = "%s"`, ctx.Status)) //nolint:gocritic // JQL syntax
	}
	if ctx.Assignee != "" {
		clauses = append(clauses, fmt.Sprintf(`assignee = "%s"`, ctx.Assignee)) //nolint:gocritic // JQL syntax
	}

	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ")
}
