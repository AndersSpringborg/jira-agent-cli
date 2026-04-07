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
		clauses = append(clauses, fmt.Sprintf(`project = "%s"`, ctx.Project))
	}
	if ctx.Epic != "" {
		clauses = append(clauses, fmt.Sprintf(`"Epic Link" = "%s"`, ctx.Epic))
	}
	if len(ctx.Labels) > 0 {
		quoted := make([]string, len(ctx.Labels))
		for i, l := range ctx.Labels {
			quoted[i] = fmt.Sprintf(`"%s"`, l)
		}
		clauses = append(clauses, fmt.Sprintf("labels in (%s)", strings.Join(quoted, ", ")))
	}
	if ctx.IssueType != "" {
		clauses = append(clauses, fmt.Sprintf(`issuetype = "%s"`, ctx.IssueType))
	}
	if ctx.Status != "" {
		clauses = append(clauses, fmt.Sprintf(`status = "%s"`, ctx.Status))
	}
	if ctx.Assignee != "" {
		clauses = append(clauses, fmt.Sprintf(`assignee = "%s"`, ctx.Assignee))
	}

	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ")
}
