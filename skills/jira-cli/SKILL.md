---
name: jira-cli
description: Drive Jira from the shell via the `jira` CLI - read, create, edit, search, assign, transition, and comment on issues; run JQL; manage sprints, boards, projects, and users. Use whenever the user mentions a Jira issue key (e.g. PROJ-123), asks to query, update, or automate Jira, or wants to look up sprints/boards/projects. All output is non-interactive JSON by default.
---

# jira-cli

Non-interactive CLI for Jira Cloud and Server. JSON by default, markdown on request, designed to be driven by an agent.

**Prerequisite:** the `jira` binary must be installed (`npm install -g @888aaen/jira-cli`) and authenticated. If `jira ping` fails, see `references/auth-setup.md`.

## Boot sequence

1. `jira auth status` - is a token present in the keychain?
2. If missing -> walk the user through `references/auth-setup.md`.
3. `jira ping` - verifies token + connectivity against the API.
4. `jira config show` - inspect the active profile and defaults.
5. Pick a command from the index below.

## Core rules

1. **JSON is the default output.** Parse it directly. Use `--format markdown` only when rendering for the user.
2. **Never invent keys.** Resolve project keys via `jira project list`, issue keys via `jira issue list` / `jira search`, account IDs via `jira user search`.
3. **JQL values use double quotes:** `status = "In Progress"`, `project = "PROJ"`.
4. **Set context once** to stop repeating flags: `jira context set --project PROJ --board-id 42`.
5. **`--profile NAME`** overrides the active profile for a single command.
6. **`jira auth status` is local-only** (keychain check). Use `jira ping` or `jira auth whoami` to confirm the token actually works.
7. **Custom fields use raw IDs:** `--field customfield_10016=5`. Repeatable.

## Intent -> command

| Intent                            | Command                                                       |
|-----------------------------------|---------------------------------------------------------------|
| List issues in current project    | `jira issue list`                                             |
| List issues assigned to me        | `jira mine` (alias `my`); `--all` includes done               |
| View an issue                     | `jira issue view PROJ-123`                                    |
| Create an issue                   | `jira issue create -p PROJ -s "Summary" -t Bug [-d ...] [--priority High]` |
| Edit summary / labels / etc.      | `jira issue edit PROJ-123 -s "New summary"`                   |
| Edit a custom field               | `jira issue edit PROJ-123 --field customfield_10016=5`        |
| Delete an issue                   | `jira issue delete PROJ-123`                                  |
| Assign to me                      | `jira issue assign PROJ-123 me`                               |
| Assign to a user                  | `jira issue assign PROJ-123 <account-id>`                     |
| Transition status                 | `jira issue move PROJ-123 "In Progress"` (case-insensitive)   |
| Comment                           | `jira issue comment PROJ-123 -b "text"`                       |
| Link issues                       | `jira issue link PROJ-1 PROJ-2 --type Blocks`                 |
| Clone                             | `jira issue clone PROJ-123`                                   |
| JQL search                        | `jira search jql "project = PROJ AND ..."`                    |
| Full-text search                  | `jira search text "terms"`                                    |
| Boards / sprints                  | `jira board list`, `jira board view 42`, `jira sprint list 42 --state active` |
| Projects                          | `jira project list`, `jira project view PROJ`                 |
| Users                             | `jira user search "jane"`                                     |
| Current user (display name)       | `jira me` (use `--raw` for full JSON)                         |
| Connectivity check                | `jira ping`                                                   |
| Open in browser                   | `jira open PROJ-123`                                          |
| Profiles                          | `jira config init/list/show/set/use/delete`                   |
| Context defaults                  | `jira context set --project PROJ` (also `--board-id`, `--epic`, `--labels`, `--display`) |

For anything not listed, run `jira <group> --help` - help is authoritative.

## Output format

| Flag                | Use                                  |
|---------------------|--------------------------------------|
| `--format json`     | default; parse programmatically       |
| `--format markdown` | structured display for the user       |

Persistent default: `jira context set --display markdown`. The `--format` flag always wins.

## When to load a reference file

- **Auth not working, first-time setup, multiple Jira instances, CI/automation tokens** -> `references/auth-setup.md`
- **Writing JQL beyond trivial queries** -> `references/jql-cookbook.md`
- **End-to-end issue lifecycle examples** -> `references/workflows.md`
