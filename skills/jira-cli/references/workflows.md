# End-to-end workflows

## Full issue lifecycle

```bash
# 1. Create
jira issue create \
  -p PROJ \
  -s "Fix login timeout" \
  -t Bug \
  -d "Users see timeout after 30s" \
  --priority High
# -> { "key": "PROJ-456", ... }

# 2. Assign to me
jira issue assign PROJ-456 me

# 3. Set a custom field (story points)
jira issue edit PROJ-456 --field customfield_10016=5

# 4. Start work
jira issue move PROJ-456 "In Progress"

# 5. Comment with findings
jira issue comment PROJ-456 -b "Root cause: connection pool exhaustion"

# 6. Resolve
jira issue move PROJ-456 "Done"
```

## Triage: bugs I should look at next

```bash
# Set context once so everything below is project-scoped
jira context set --project PROJ --display markdown

# Open bugs assigned to me, sorted by priority
jira search jql \
  'assignee = currentUser() AND issuetype = Bug AND status != Done ORDER BY priority DESC, updated DESC'

# Or, for everything on my plate:
jira mine
```

## Sprint kickoff: what's in the active sprint?

```bash
# Find the active sprint on board 42
jira sprint list 42 --state active

# Then list its issues via JQL (substitute the sprint name)
jira search jql 'sprint = "Sprint 17" AND project = PROJ'
```

## Bulk transition from a JQL query

```bash
# Move every "Ready for QA" issue assigned to me into "In QA"
jira search jql 'assignee = currentUser() AND status = "Ready for QA"' \
  | jq -r '.[].key' \
  | xargs -I{} jira issue move {} "In QA"
```

## Cloning a template issue

```bash
jira issue clone PROJ-100
# -> returns the new issue key; edit it as needed
jira issue edit PROJ-512 -s "Onboarding: new hire 2026-Q3"
```
