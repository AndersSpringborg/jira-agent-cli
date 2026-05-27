# JQL cookbook

All examples assume the active context has `--project PROJ` set, or substitute your own key. JQL string values use double quotes; the surrounding shell string uses single quotes to avoid escaping.

## Common patterns

```bash
# Issues assigned to current user
jira search jql 'assignee = currentUser() AND project = PROJ'

# Open bugs
jira search jql 'project = PROJ AND issuetype = Bug AND status != Done'

# Updated in the last 7 days
jira search jql 'project = PROJ AND updated >= -7d ORDER BY updated DESC'

# In a specific sprint
jira search jql 'sprint = "Sprint 5" AND project = PROJ'

# Unassigned
jira search jql 'project = PROJ AND assignee is EMPTY'

# Mentions me
jira search jql 'text ~ currentUser()'

# Created by me this quarter
jira search jql 'reporter = currentUser() AND created >= startOfQuarter()'

# Linked to a specific issue
jira search jql 'issue in linkedIssues("PROJ-123")'
```

## Useful operators & functions

- Comparison: `=`, `!=`, `>`, `>=`, `<`, `<=`, `in (...)`, `not in (...)`
- Text: `~` (contains), `!~` (does not contain)
- Null: `is EMPTY`, `is not EMPTY`
- Time math: `-1d`, `-2w`, `-1M`; functions like `startOfDay()`, `endOfWeek()`, `startOfMonth()`
- User: `currentUser()`, `membersOf("group-name")`

## Sorting & limiting

`ORDER BY` is part of JQL. To cap results, use the CLI flag:

```bash
jira search jql 'project = PROJ ORDER BY created DESC' --max 25
```

## Tips

- If a field name contains spaces or special characters, wrap it in double quotes: `"Epic Link" = PROJ-1`.
- Custom fields can be referenced by name (`"Story Points"`) or ID (`cf[10016]`).
- Use `jira search text "terms"` for fuzzy full-text search instead of constructing `text ~ "..."` JQL.
