# jira-cli

A non-interactive CLI for Jira built for AI agents, CI, and automation. All output is machine-readable JSON by default, so it drops straight into `jq` pipelines, shell scripts, and LLM tool calling.

```bash
jira search jql "assignee = currentUser() AND status != Done" | jq -r '.[].key'
```

## Why use this

- **JSON by default, clean streams.** Results go to `stdout`, all diagnostics to `stderr`, and failures return a non-zero exit code. `jq`, `rg`, and `wc` work without choking on error blobs, and scripts check `$?` instead of parsing strings.
- **Auth stays out of band.** You log in once; the token lives in the OS keychain and is never written to disk or exposed to an agent. Execution inherits an already-authenticated environment.
- **Inspectable, scoped state.** `jira context set work --project PROJ --profile default` saves project and authentication defaults together; `jira context use work` switches scope explicitly.
- **Lean output.** Project only the fields you need with repeatable `-F customfield_x` flags, keeping responses (and token usage) small.
- **No daemon, no custom protocol.** It's a plain binary. Every action is one replayable command line — debug by copying it from your shell history and running it again.
- **Server-side audit.** `jira me audit --date YYYY-MM-DD` reconstructs what actually changed on a given day from Jira's own changelog, independent of local state.

See [What makes this an agent CLI](#what-makes-this-an-agent-cli) for the longer rationale.

## Why not the existing jira-cli?

This project is inspired by [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli) — a feature-rich **interactive** Jira command line with a full TUI (tables, keyboard navigation, prompts). Think of it as the [k9s](https://k9scli.io/) of Jira: visual and built for humans at a terminal.

This takes the opposite approach — the **kubectl** of Jira: non-interactive, scriptable, and built for agents and automation. No TUI, no prompts, just structured output machines can parse. Want a great interactive experience? Use that one. Want to wire Jira into an AI agent, CI pipeline, or shell script? Use this.


## Install

### npm (recommended)

```bash
npm install -g @888aaen/jira-cli
```

Installs the `jira` binary for macOS (arm64, x64), Linux (x64, arm64), and Windows (x64).

### Build from source

**Prerequisites:** [Go 1.25+](https://go.dev/dl/)

```bash
git clone https://github.com/AndersSpringborg/jira-agent-cli.git
cd jira-agent-cli
sudo make install      # builds and installs to /usr/local/bin/jira
sudo make uninstall    # to remove
```

## Quick Start (AI agent)

Give an agent Jira access in two commands:

```bash
npm install -g @888aaen/jira-cli
npx skills@latest add AndersSpringborg/jira-agent-cli
```

The first installs the `jira` binary. The second installs the [jira-cli skill](skills/jira-cli/SKILL.md) into `~/.claude/skills/jira-cli/`, so any Claude Code agent on the machine learns to drive it — it checks for an existing session, guides you through login if needed, and picks the right command for each request.

## Quick Start (manual)

### 1. Authenticate

Create an API token at https://id.atlassian.com/manage-profile/security/api-tokens, then:

```bash
jira auth login --server https://your-org.atlassian.net \
  --email you@example.com --token YOUR_API_TOKEN
```

The token is stored in the OS keychain — never written to disk.

> **Jira Cloud and Server/Data Center.** Cloud profiles (`*.atlassian.net`) use
> basic auth + REST v3 + ADF bodies. Server/Data Center profiles use PAT/bearer
> auth + REST v2 + wiki/plain text bodies.

### 2. Verify and set defaults

```bash
jira ping
jira context set work --project PROJ --board-id 42 --profile default
jira context use work                  # switch project and authentication profile together
jira context list                      # inspect all saved contexts
```

A per-command `--format` flag always overrides the context default. See [Output Formats](#output-formats).

### 3. Use it

```bash
jira issue list                                          # issues in your project
jira issue view PROJ-123                                  # includes attachment metadata
jira issue attachment download PROJ-123 10042 -o /tmp/screenshot.png
jira issue create -p PROJ -s "Fix login bug" -t Bug
jira search jql "project = PROJ AND status = 'In Progress'"
jira issue list | jq '.[].key'
jira issue ready                       # unresolved work with no unresolved blockers
jira issue graph | jq '{ready, cycles}'
jira issue graph-pretty                # visual dependency check for humans
```

## Output Formats

| Flag                | Description                              |
|---------------------|------------------------------------------|
| `--format json`     | Machine-readable JSON (default)          |
| `--format markdown` | Structured markdown, fewer tokens for LLMs |

Set a persistent default on a named context with `jira context set work --display markdown`; `--format` always overrides it.

## Debugging Jira responses

Use the global `--debug` flag to print the raw Jira HTTP response to `stderr` while keeping normal command output on `stdout`:

```bash
jira ping --debug 2>jira-debug.log
jira issue view PROJ-123 --debug 2>jira-debug.log
```

The debug log includes the request method and URL, response status, response headers, and unparsed response body. This is useful when an on-prem proxy or login page returns HTML and the command fails with an error such as `invalid character '<'`. Authentication request headers are never printed, and `Set-Cookie` response headers are redacted. Response bodies can still contain sensitive Jira data, so review the file before sharing it.

## Writing to Jira

Every mutation is a single command with explicit flags, so the transcript line is exactly what changed. Write commands print the result as JSON and exit non-zero on failure, so you can chain them with `&&` and check the exit code.

Issue descriptions and comment bodies are written in **Markdown** (CommonMark): headings, bullet/ordered lists, fenced code blocks, links, bold/italic. The CLI converts it to the format the server expects — ADF on Jira Cloud, wiki markup on Server/Data Center — so you never hand-write `{code}` or `h3.` wiki syntax.

```bash
# Create (prints structured JSON with .key; add --raw for the full Jira response)
jira issue create -p PROJ -s "Fix login bug" -t Bug -b "Steps to reproduce..."

# Edit fields, including custom fields by id (-F is repeatable)
jira issue edit PROJ-123 -s "New summary" -l backend -F customfield_10145="value"

# Transition through the workflow
jira issue move PROJ-123 "In Progress"
jira issue move PROJ-123 Done --resolution Fixed --comment "Shipped in v1.2"

# Assign and comment ('me' for yourself, 'x' to unassign; otherwise pass an account ID,
# username, or email resolvable by `jira user search`)
jira issue assign PROJ-123 me
jira issue comment add PROJ-123 "Investigated -- root cause was a stale cache."

# Link, inspect dependencies, clone, delete
jira issue link PROJ-123 PROJ-456 "blocks"
jira issue ready                       # highest-leverage ready work first
jira issue graph                       # nodes, blocker -> blocked edges, ready, blocked, cycles
jira issue graph-pretty                # connected human-readable dependency view
jira issue clone PROJ-123 -s "Follow-up: ..."
jira issue delete PROJ-123

# Sprint management
jira sprint add 42 PROJ-123 PROJ-456
```

Capture a created key and act on it in the same script:

```bash
key=$(jira issue create -p PROJ -s "Automated task" -t Task | jq -r '.key')
jira issue move "$key" "In Progress" && jira issue assign "$key" me
```

Run `jira issue <verb> --help` for the full flag set on any command.

### Inspecting attachments

`jira issue get PROJ-123` (alias of `view`) includes attachment metadata by default, including each attachment's ID, filename, media type, size, and Jira content URL. JSON keeps Jira's original attachment objects; markdown renders a dedicated attachments table. If you explicitly narrow fields with `--field` or `--fields`, include `attachment` yourself.

Download binary content to a file before asking an LLM or another local tool to inspect it:

```bash
# IDs are unambiguous and preferred for automation
jira issue attachment download PROJ-123 10042 --output /tmp/screenshot.png

# An exact filename also works when only one attachment has that name
jira issue attachment download PROJ-123 screenshot.png -o /tmp/screenshot.png
```

The command never prints binary data to stdout. It downloads through the authenticated Jira client, writes through a temporary file so failures do not leave a partial destination, and returns structured metadata including the local absolute `path`. Cross-origin redirects do not receive Jira credentials.

## Commands

| Command        | Description                                  |
|----------------|----------------------------------------------|
| `jira api`     | Raw access to any Jira REST endpoint; `--list` to discover endpoints |
| `jira auth`    | Login, logout, status, whoami                |
| `jira config`  | Manage profiles (init, list, show, set, use, delete) |
| `jira context` | Set default filters (project, board, labels, etc.)   |
| `jira issue`   | Full issue lifecycle plus dependency analysis (`ready`, `graph`) |
| `jira board`   | List boards, view board issues               |
| `jira sprint`  | List, start, close sprints; add issues       |
| `jira project` | List and view projects                       |
| `jira search`  | JQL and full-text search                     |
| `jira user`    | Search and get users                         |
| `jira me`      | Show current user; `me audit` for daily activity |
| `jira mine`    | List issues assigned to you                  |
| `jira open`    | Open project or issue in browser             |
| `jira ping`    | Check connectivity to Jira                   |

Run `jira <command> --help` for details on any command. Failed commands also print the failing command's complete help text to stderr, so agents can recover without a second discovery call.

### Raw API access

`jira api` is the escape hatch when no predefined command covers what you need — raw SQL for Jira. It sends any request with the active profile's authentication and prints the response body verbatim (pipe to `jq`):

```bash
# GET is the default; full paths pass through verbatim (platform, agile, plugins)
jira api /rest/agile/1.0/board/42/backlog
jira api "/rest/api/3/issue/PROJ-1?expand=changelog,renderedFields" | jq .changelog

# shorthand paths get the platform prefix for your instance flavor:
# /rest/api/3 on Jira Cloud, /rest/api/2 on Jira Server/Data Center
jira api issue/PROJ-1

# writes: -d implies POST; @file and - (stdin) work too; -X and -H as in curl
jira api issue -d '{"fields":{...}}'
echo '{"issues":["PROJ-1"]}' | jira api /rest/agile/1.0/sprint/7/issue -d -
```

`jira api --list [filter...]` browses the API surface itself — endpoint catalogs generated from Atlassian's official OpenAPI specs are embedded in the binary, so it works offline and without authentication. The catalog matches your instance flavor (Cloud v3 vs Server/DC v2), deduced from the active profile:

```bash
jira api --list sprint            # every sprint-related endpoint
jira api --list "POST worklog"    # filter terms match method, path, and description
```

Failed requests print Jira's error body (that's the useful part) and exit non-zero.

### Resolving issue dependencies

Jira's `Blocks` links form a directed graph from blocker to blocked issue. The dependency commands use the active project/context filters and accept the same scope flags (`--project`, `--status`, `--assignee`, `--type`, `--epic`, `--label`, and `--max`). Status accepts a comma-separated set, for example `--status "Define,To Do,Backlog"`. Use `--link-type` if your Jira site names dependency links differently.

```bash
jira issue ready --project PROJ
jira issue graph --project PROJ | jq '.edges'
jira issue graph --project PROJ --format markdown
jira issue graph-pretty --project PROJ
```

`issue ready` returns unresolved issues with no unresolved blockers, ordered by how much unresolved work they directly unblock. A blocker is resolved only when its Jira `resolution` field is set. `issue graph` returns a stable object containing `nodes`, directed `edges`, `ready`, `blocked`, and `cycles`. Linked blockers outside the selected scope remain in the graph with `inScope: false`.

`issue graph-pretty` builds the complete graph before rendering it as deterministic plain text. It condenses cycles, assigns dependency-depth layers, and draws each connected component from top to bottom. Continuous lines make branches and shared downstream joins visible without duplicating subtrees. Compact issue keys keep larger graphs readable; use `issue graph` when summaries or full node metadata are needed. `graph-pretty` always emits plain text regardless of `--format`.

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

## Configuration

Config lives at `~/.config/jira-cli/config.yml`. You normally don't edit it by hand — use the `jira config` and `jira context` commands.

### Profiles

Manage multiple Jira instances:

```bash
jira config init --profile work --base-url https://work.atlassian.net
jira auth login --profile work --server https://work.atlassian.net \
  --email you@work.com --token YOUR_TOKEN
jira config use work                 # switch default
jira issue list --profile work       # use a profile for one command
```

### Environment Variables

These override config file values and are useful in CI/automation:

| Variable          | Description                       |
|-------------------|-----------------------------------|
| `JIRA_BASE_URL`   | Jira server URL                   |
| `JIRA_TOKEN`      | API token (bypasses OS keychain)  |
| `JIRA_EMAIL`      | User email                        |
| `JIRA_AUTH_TYPE`  | Auth type: `basic` or `pat`       |
| `JIRABOT_PROFILE` | Profile name to use               |

## What makes this an agent CLI

### Out-of-band authentication
Authentication and execution are decoupled. A human or CI process runs `jira auth login` once; the token is stored in the OS keychain, never on disk. An agent never sees, requests, or routes the credential — it inherits an already-authenticated environment, and access is revoked system-side without any model trust.

### Inspectable state and blast-radius control
`jira context set work --project PROJ --board-id 42 --profile default` writes project filters and the authentication profile to a named context. `jira context use work` makes that context active, while `jira context list` keeps every saved scope inspectable. An explicit per-command `--profile` still overrides the context's authentication profile.

### Built for the Unix pipe
The CLI leans on standard tools (`jq`, `rg`, `grep`, `wc`) instead of a bespoke processing engine. `stdout` carries only the result; diagnostics, warnings, and errors go to `stderr`, so pipelines never choke on error output. Failed calls return a non-zero exit code, so success/failure is read from `$?` rather than parsed from text.

```bash
jira search jql "assignee = currentUser()" | jq -r '.[].key'
jira issue list | jq '[.[] | select(.status.name=="Done")] | length'
jira issue view CER-1 --raw | rg -o '"customfield_\d+"'
```

### Non-mutating discovery and field projection
Read paths (`search jql`, `search text`, `issue list`, `mine`, `issue view`, attachment downloads) do not mutate Jira and return consistent, keyed JSON (`key`, `fields.*`). `issue view` includes attachment metadata by default, and `issue attachment download` materializes selected binary content at an explicit local path for inspection. Custom-field projection keeps responses small so an agent retrieves only what it needs via repeatable `--field`/`-F` flags; `issue view` also keeps `--fields` as a comma-separated alias.

### CLI over MCP
Rather than running a Model Context Protocol server, this is a standard binary: no long-lived daemon, no open socket, no persistent state between executions. Every action is the exact execution string, so debugging is just re-running the command from shell history, and agents reuse shell operators (`&&`, `||`, `|`, `>`) instead of learning a custom RPC protocol.

### Dual-layer auditing
Verification rests on hard records, not transcripts. Shell history is a replayable record of *what was attempted* — every input is a command-line argument, not hidden state. `jira me audit --date YYYY-MM-DD` queries Jira's server-side changelog to reconstruct *what actually mutated* on a given day for the authenticated user, independent of local terminal state.
