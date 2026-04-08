---
name: jira-cli
description: "Use jira-cli to interact with Jira: issue CRUD, JQL search, sprint and board management, project operations, and user lookup. All commands are non-interactive with JSON output by default. Load this skill whenever the user needs to read, create, update, or search Jira issues, manage sprints, or automate Jira workflows."
compatibility: Requires the jira binary (npm install -g @888aaen/jira-cli)
metadata:
  author: 888aaen
  cli-help: "jira --help"
---

# jira-cli

> **Prerequisite:** The CLI must be installed (`npm install -g @888aaen/jira-cli`) and authenticated before use.

## Agent Execution Order

1. **Check auth** -- run `jira auth status` to verify a token exists
2. **If not authenticated** -- guide the user through authentication (see below)
3. **Verify connectivity** -- run `jira ping` (makes a real API call)
4. **Check context** -- run `jira config show` to see default project/board
5. **Execute task** -- use the command index below to pick the right command

## Core Rules

1. **All output is non-interactive JSON by default** -- never expect prompts or interactive input
2. **Use `--format json`** (default) when parsing output programmatically; use `--format markdown` when presenting to the user
3. **Set context defaults** to avoid repeating flags: `jira context set --project PROJ --board-id 42`
4. **JQL strings use double quotes** for values: `status = "In Progress"`, `project = "PROJ"`
5. **The `--profile` flag** overrides the default profile for any single command
6. **`jira ping` validates auth + connectivity**; `jira auth status` only checks local keychain
7. **Never guess issue keys or project keys** -- always get them from `jira issue list`, `jira project list`, or user input

## Authentication Setup

### Jira Cloud (*.atlassian.net)

The user must create an API token. Direct them to:
https://id.atlassian.com/manage-profile/security/api-tokens

Then run:

```bash
jira config init --base-url https://your-org.atlassian.net
jira auth login \
  --server https://your-org.atlassian.net \
  --email USER_EMAIL \
  --token API_TOKEN
```

### Jira Server / Data Center (PAT)

The user creates a Personal Access Token in Jira (Profile > Personal Access Tokens), then:

```bash
jira config init --base-url https://jira.example.com
jira auth login \
  --server https://jira.example.com \
  --token PERSONAL_ACCESS_TOKEN
```

Auth type is auto-detected from the URL (`.atlassian.net` = basic, otherwise = PAT).
Tokens are stored in the OS keychain -- never written to disk.

## Output Formats

| Flag                | Description                           |
|---------------------|---------------------------------------|
| `--format json`     | Machine-readable JSON (default)       |
| `--format markdown` | Structured markdown for LLM display   |

Set a persistent default: `jira context set --display markdown`

The `--format` flag always overrides the context default.

## Intent -> Command Index

| Intent                          | Command                                    | Notes                                    |
|---------------------------------|--------------------------------------------|------------------------------------------|
| List issues in a project        | `jira issue list`                          | Uses context project if set              |
| View a specific issue           | `jira issue view PROJ-123`                 | Returns full issue detail                |
| Create an issue                 | `jira issue create -p PROJ -s "Summary" -t Bug` | `-d` for description, `--priority` for priority |
| Edit an issue                   | `jira issue edit PROJ-123 -s "New summary"` | Supports labels, components, fixVersions |
| Delete an issue                 | `jira issue delete PROJ-123`               |                                          |
| Assign an issue to me           | `jira issue assign PROJ-123 me`              | Resolves current user automatically       |
| Assign an issue to a user       | `jira issue assign PROJ-123 <account-id>`    | Use `jira user search` to find account ID |
| Move issue status               | `jira issue move PROJ-123 "In Progress"`   | Matches transition name (case-insensitive) |
| Add a comment                   | `jira issue comment PROJ-123 -b "text"`    |                                          |
| Link two issues                 | `jira issue link PROJ-1 PROJ-2 --type Blocks` |                                       |
| Clone an issue                  | `jira issue clone PROJ-123`                |                                          |
| Search with JQL                 | `jira search jql "project = PROJ AND ..."`  | Full JQL support                         |
| Full-text search                | `jira search text "search terms"`          | Searches across issue text fields        |
| List boards                     | `jira board list`                          |                                          |
| View board issues               | `jira board view 42`                       | Board ID from `jira board list`          |
| List sprints                    | `jira sprint list 42`                      | `--state active/closed/future`           |
| List projects                   | `jira project list`                        |                                          |
| View project details            | `jira project view PROJ`                   |                                          |
| Search users                    | `jira user search "jane"`                  |                                          |
| Get current user                | `jira me`                                  | Script-friendly: `jira me` returns display name |
| Check connectivity              | `jira ping`                                | Validates auth + API access              |
| Open in browser                 | `jira open PROJ-123`                       | Opens issue or project in default browser |
| Manage profiles                 | `jira config init/list/show/set/use/delete` |                                         |
| Set context defaults            | `jira context set --project PROJ`          | Also: `--board-id`, `--epic`, `--labels` |
| Check auth status               | `jira auth status`                         | Local keychain check only                |
| Show authenticated user         | `jira auth whoami`                         | Makes API call, returns user details     |

## Common JQL Patterns

```bash
# Issues assigned to current user
jira search jql "assignee = currentUser() AND project = PROJ"

# Open bugs in a project
jira search jql "project = PROJ AND issuetype = Bug AND status != Done"

# Updated in the last 7 days
jira search jql "project = PROJ AND updated >= -7d ORDER BY updated DESC"

# Issues in a specific sprint
jira search jql "sprint = 'Sprint 5' AND project = PROJ"

# Unassigned issues
jira search jql "project = PROJ AND assignee is EMPTY"
```

## Issue Lifecycle Example

```bash
# Create
jira issue create -p PROJ -s "Fix login timeout" -t Bug -d "Users see timeout after 30s" --priority High

# Assign
# Assign to myself
jira issue assign PROJ-456 me

# Move to In Progress
jira issue move PROJ-456 "In Progress"

# Add a comment
jira issue comment PROJ-456 -b "Root cause identified: connection pool exhaustion"

# Move to Done
jira issue move PROJ-456 "Done"
```

## Context System

The context system stores defaults so you don't repeat flags:

```bash
jira context set --project PROJ        # default project
jira context set --board-id 42         # default board
jira context set --display markdown    # default output format
```

After setting context, `jira issue list` automatically uses project PROJ.

## Multiple Profiles

```bash
# Create a second profile
jira config init --profile work --base-url https://work.atlassian.net
jira auth login --profile work --server https://work.atlassian.net --email you@work.com --token TOKEN

# Switch default
jira config use --profile work

# One-off command with a different profile
jira issue list --profile personal
```

## Environment Variables (CI/Automation)

| Variable          | Description                       |
|-------------------|-----------------------------------|
| `JIRA_BASE_URL`   | Jira server URL                  |
| `JIRA_TOKEN`      | API token (bypasses OS keychain) |
| `JIRA_EMAIL`      | User email                       |
| `JIRA_AUTH_TYPE`   | Auth type: `basic` or `pat`     |
| `JIRABOT_PROFILE`  | Profile name to use             |
