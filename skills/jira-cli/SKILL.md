---
name: jira-cli
description: Drive Jira from the shell via the `jira` CLI - read, create, edit, search, assign, transition, and comment on issues; run JQL; manage sprints, boards, projects, and users. Use whenever the user mentions a Jira issue key (e.g. PROJ-123), asks to query, update, or automate Jira, or wants to look up sprints/boards/projects. All output is non-interactive JSON by default.
---

# jira-cli

Non-interactive CLI for Jira Cloud and Jira Server/Data Center. JSON by default, markdown on request, designed to be driven by an agent. Cloud uses REST v3 + ADF; Server/Data Center uses REST v2 + wiki/plain text bodies.

**Prerequisite:** the `jira` binary must be installed (`npm install -g @888aaen/jira-cli`) and authenticated. If `jira ping` fails, see `references/auth-setup.md`.

## Boot sequence

1. `jira auth status` - is a token present in the keychain?
2. If missing -> the user logs in themselves; point them to `references/auth-setup.md`. Never handle the token yourself.
3. `jira ping` - verifies token + connectivity against the API.
4. `jira config show` - inspect the active profile and defaults.
5. Pick a command from the index below.

## Draft before writing (human approval required)

**Never run a mutating command without showing the user the exact command and its effect, then getting their approval.** Covers `issue create`, `edit`, `delete`, `assign`, `move`, `comment add`, `link` / `unlink`, `clone`, and `sprint add` / `start` / `close`.

Flow: **draft -> approve -> execute.**

1. **Draft.** Show the exact command line(s) and a one-line summary of the effect.
2. **Approve.** Wait for confirmation. If they want changes, redraft and show it again — never execute a partially-approved change.
3. **Execute.** Run the approved command verbatim. If a key won't resolve or a field is rejected, stop and re-draft — don't improvise a different write.

Reads (`list`, `view`, `search`, `me`/`mine`, `ping`, board/project/user queries) never mutate — run them freely, including to gather data for an accurate draft.

## Core rules

1. **Draft mutating commands for approval before running them.** See [Draft before writing](#draft-before-writing-human-approval-required). Reads run freely.
2. **JSON is the default output.** Parse it directly. Use `--format markdown` only when rendering for the user.
3. **Never invent keys.** Resolve project keys via `jira project list`, issue keys via `jira issue list` / `jira search`, account IDs via `jira user search`.
4. **JQL values use double quotes:** `status = "In Progress"`, `project = "PROJ"`.
5. **Set and activate a named context once** to stop repeating flags: `jira context set work --project PROJ --board-id 42 --profile default && jira context use work`.
6. **`--profile NAME`** overrides the active context's authentication profile for a single command.
7. **`jira auth status` is local-only** (keychain check). Use `jira ping` or `jira auth whoami` to confirm the token actually works.
8. **Custom fields use raw IDs:** `--field customfield_10016=5`. Repeatable.
9. **Bodies are Markdown.** `--body`/`-b` (and comment text) take CommonMark - headings, lists, fenced code blocks, links, bold/italic. The CLI converts it to ADF on Cloud and wiki markup on Server/DC. Write Markdown, not raw `{code}`/`h3.` wiki syntax.

## Intent -> command

| Intent                            | Command                                                       |
|-----------------------------------|---------------------------------------------------------------|
| List issues in current project    | `jira issue list` (alias `ls`)                                |
| Find dependency-ready work        | `jira issue ready` (no unresolved `Blocks` blockers)          |
| Inspect dependency graph          | `jira issue graph` (`nodes`, blocker -> blocked `edges`, `ready`, `blocked`, `cycles`) |
| Visually verify dependencies      | `jira issue graph-pretty` (connected lines and concise state markers) |
| List issues assigned to me        | `jira mine` (alias `my`); `--all` includes done               |
| View an issue                     | `jira issue view PROJ-123` (alias `get`; supports `-F`)       |
| Create an issue                   | `jira issue create -p PROJ -s "Summary" -t Bug [-b "body"] [-y High]` |
| Edit summary / labels / etc.      | `jira issue edit PROJ-123 -s "New summary"` (alias `update`)  |
| Edit a custom field               | `jira issue edit PROJ-123 --field customfield_10016=5`        |
| Delete an issue                   | `jira issue delete PROJ-123`                                  |
| Assign to me                      | `jira issue assign PROJ-123 me` (`x` to unassign)             |
| Assign to a user                  | `jira issue assign PROJ-123 <account-id-or-email>`            |
| Transition status                 | `jira issue move PROJ-123 "In Progress"` (alias `transition`, case-insensitive) |
| Comment                           | `jira issue comment add PROJ-123 "text"`                      |
| Link / unlink                     | `jira issue link PROJ-1 PROJ-2 Blocks`, `jira issue unlink PROJ-1 PROJ-2` |
| Clone                             | `jira issue clone PROJ-123`                                   |
| JQL search                        | `jira search jql "project = PROJ AND ..."`                    |
| Full-text search                  | `jira search text "terms"`                                    |
| Boards                            | `jira board list`, `jira board get 42`, `jira board issues 42` |
| Sprints                           | `jira sprint list 42 --state active`, `jira sprint get <id>`, `jira sprint issues <id>` |
| Projects                          | `jira project list`, `jira project get PROJ`                  |
| Users                             | `jira user search "jane"`, `jira user get <account-id>`       |
| Current user (display name)       | `jira me` (use `--raw` for full JSON)                         |
| My activity for a day             | `jira mine audit [--date YYYY-MM-DD]` (also `jira me audit`)   |
| Connectivity check                | `jira ping`                                                   |
| Update the CLI                    | `jira update` (uses npm for npm installations, GitHub otherwise) |
| Open in browser                   | `jira open PROJ-123`                                          |
| Profiles                          | `jira config init/list/show/set/use/delete`                   |
| Anything with no command above    | `jira api <path>` — raw REST passthrough (`-X`, `-d`, `-H`)    |
| Discover API endpoints            | `jira api --list [filter...]` (offline, no auth needed)        |
| Context defaults                  | `jira context set NAME --project PROJ --profile PROFILE`; switch with `jira context use NAME`, inspect with `jira context list` |

When no predefined command fits, use `jira api`: it sends a raw request with the profile's auth and prints the response verbatim. Full paths (starting with `/`) pass through unchanged; shorthand paths like `issue/PROJ-1` get the platform prefix for the instance flavor (`/rest/api/3` on Cloud, `/rest/api/2` on Server/Data Center). `-d` implies POST (`@file` and `-` for stdin work); find the right endpoint first with `jira api --list <terms>`, which searches an embedded catalog of the official API for the profile's flavor. Errors print Jira's error body and exit non-zero.

For anything not listed, run `jira <group> --help` - help is authoritative. Failed commands automatically include the failing command's full help text on stderr; use it to correct the next call instead of repeating the same command.

Dependency commands inherit the active project/context filters and accept explicit scope flags; `--status "Define,To Do,Backlog"` selects multiple statuses. `jira issue ready` orders actionable issues by how many unresolved issues they directly unblock. `jira issue graph` retains linked blockers outside the selected scope as `inScope: false`; inspect `.cycles` before trying to sequence cyclic work. Use `jira issue graph-pretty` when a human should verify the full topology: it builds the graph first, layers it by dependency depth, and draws continuous top-down branch and join lines. `●`, `○`, `✓`, `◇`, and `↻` mark ready, blocked, resolved, external, and cyclic nodes. A blocker counts as resolved only when Jira's `resolution` field is set. Use `--link-type` for a custom dependency link name.

Example `graph-pretty` output:

```text
Dependency graph (blocker ──▶ blocked)
Legend: ● ready  ○ blocked  ✓ resolved  ◇ external  ↻ cycle

Component 1
          ● PROJ-1
              │
       ┌──────┴──────┐
       ▼             ▼
   ○ PROJ-2      ○ PROJ-3
       │             │
       └──────┬──────┘
              ▼
          ○ PROJ-4
```

## Output format

| Flag                | Use                                  |
|---------------------|--------------------------------------|
| `--format json`     | default; parse programmatically       |
| `--format markdown` | structured display for the user       |

Persistent default: `jira context set NAME --display markdown`. The `--format` flag always wins.

## When to load a reference file

- **Auth not working, first-time setup, multiple Jira instances, CI/automation tokens** -> `references/auth-setup.md`
- **Writing JQL beyond trivial queries** -> `references/jql-cookbook.md`
- **End-to-end issue lifecycle examples** -> `references/workflows.md`

## Reporting CLI problems

If the `jira` CLI itself misbehaves (a crash, a wrong result, a missing flag — not a Jira data error), and `gh` is installed (`gh --version` succeeds) and authenticated, offer to file a bug. Get the user's OK first, then:

```bash
gh issue create --repo AndersSpringborg/jira-agent-cli \
  --title "<short summary>" \
  --body "<exact command run, expected vs actual, jira --version, OS>"
```

If `gh` is missing or not authenticated, give the user the title and body to file manually at https://github.com/AndersSpringborg/jira-agent-cli/issues. Never paste tokens or secrets into an issue.
