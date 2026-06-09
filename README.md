# jira-cli

A non-interactive CLI for Jira designed for AI agents and automation. All output is machine-readable JSON by default, making it ideal for `jq` pipelines and LLM tool calling.

## Credit

This project is inspired by [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli) -- a feature-rich **interactive** Jira command line with a full TUI (tables, keyboard navigation, interactive prompts). Think of it as the [k9s](https://k9scli.io/) of Jira: powerful, visual, and built for humans at a terminal.

This project takes a different approach. It is the **kubectl** of Jira: non-interactive, scriptable, and designed for AI agents. No TUI, no prompts -- just structured output that machines can parse. If you want a great interactive experience, use [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli). If you want to wire Jira into an AI agent, CI pipeline, or shell script, use this.

## What makes this an "Agent CLI"

### 1. Out-of-band Authentication
Authentication and execution are strictly decoupled.
* **Mechanism:** A human or CI process runs `jira auth login` once. The token is stored securely in the **OS keychain, never to disk**.
* **Security Guarantee:** The LLM agent never sees, requests, handles, or routes credential strings. It inherits an already-authenticated environment. Access is revoked system-side, requiring no model trust.

### 2. Inspectable State & Blast Radius Controls
State is managed via disk, not LLM context windows.
* **Scope:** `jira context set --project PROJ --board-id 42` establishes local default parameters.
* **Blast Radius:** This restricts the agent's default operating scope without requiring the LLM to continuously append `--project` to every call.
* **Overrides:** The context is explicit, inspectable state. If the agent needs to break scope, it must do so explicitly via per-command overrides (e.g., `--profile`).

### 3. Unix Pipe Compliance (JSON by Default)
The CLI relies on standard shell utilities (`jq`, `rg`, `grep`, `wc`) rather than building a bespoke internal processing engine or requiring SDKs.
* **Strict Stream Separation:** `stdout` emits **only** machine-readable JSON. All diagnostics, warnings, and errors are routed to `stderr`. This guarantees pipeline tools like `jq` will not choke on error blobs.
* **Reliable Exit Codes:** Failed API calls or validations result in a non-zero exit code. Agents detect success/failure via the `$?` status code natively, avoiding the need to parse output strings to determine operation state.

```bash
# Extract issue keys natively
jira search jql "assignee = currentUser()" | jq -r '.[].key'            

# Filter and aggregate
jira issue list | jq '[.[] | select(.status.name=="Done")] | length'    

# Inspect schemas
jira issue view CER-1 --raw | rg -o '"customfield_\d+"'                 
```
*(Note: `--format markdown` is supported solely for human-readable summaries, but is not the default path).*

### 4. Idempotent Discovery & Field Projection
Read paths are explicitly non-mutating and structured for fanning out searches.
* **Stable Schemas:** Operations like `search jql`, `search text`, and `issue view` return consistent, keyed JSON objects (`key`, `fields.*`).
* **Targeted Retrieval:** Agents can project exact custom fields into any read path via repeatable flags (`--field customfield_x` or `-F`). This ensures the agent retrieves only the data necessary for its reasoning loop, minimizing token overhead.

### 5. CLI over MCP (Stateless vs. Stateful)
Instead of implementing a Model Context Protocol (MCP) server, this tool utilizes a standard CLI binary.
* **Zero Standing Surface:** There is no long-lived daemon, no open socket, and no persistent state between executions.
* **Direct Replayability:** The interface is the exact execution string. To debug an agent, a human simply copies the exact command from the shell history and executes it.
* **Native Composability:** Agents already understand shell operators (`&&`, `||`, `|`, `>`). A CLI leverages existing agent capabilities instead of requiring the LLM to learn a custom RPC protocol.

### 6. Dual-Layer Auditing
Verification relies on hard records, not LLM transcripts.
* **Local Audit (Execution):** The shell history acts as the immutable log of *what was attempted*. The inputs are explicit command-line arguments.
* **Remote Audit (Mutation):** `jira me audit` (or `jira mine audit --date YYYY-MM-DD`) queries Jira's server-side changelogs. It reconstructs exactly *what mutated* on a given day based on the authenticated user's activity. This provides an independent, cryptographic-equivalent verification of the agent's actual impact, regardless of local terminal state.

## Install

### npm (recommended)

```bash
npm install -g @888aaen/jira-cli
```

This installs the `jira` binary for your platform. Works on macOS (arm64, x64), Linux (x64, arm64), and Windows (x64).

### Build from source

**Prerequisites:** [Go 1.25+](https://go.dev/dl/)

```bash
git clone https://github.com/AndersSpringborg/jira-agent-cli.git
cd jira-agent-cli
sudo make install
```

This builds the binary and copies it to `/usr/local/bin/jira`.

To uninstall:

```bash
sudo make uninstall
```

## Quick Start (AI Agent)

Give your AI agent Jira superpowers in two commands:

```bash
npm install -g @888aaen/jira-cli
npx skills@latest add AndersSpringborg/jira-agent-cli
```

The first installs the `jira` binary. The second installs the [jira-cli skill](skills/jira-cli/SKILL.md) into `~/.claude/skills/jira-cli/` so any Claude Code agent on the machine learns how to drive it.

After adding the skill, the agent will:
1. Check for an existing Jira auth session
2. Guide you through login if needed
3. Use the right `jira` command for any Jira-related request

## Quick Start

### 1. Authenticate

**Jira Cloud** (*.atlassian.net):

1. Create an API token at https://id.atlassian.com/manage-profile/security/api-tokens
2. Run:

```bash
jira auth login \
  --server https://your-org.atlassian.net \
  --email you@example.com \
  --token YOUR_API_TOKEN
```

**Jira Server / Data Center** (Personal Access Token):

1. In Jira, go to Profile > Personal Access Tokens
2. Run:

```bash
jira auth login \
  --server https://jira.example.com \
  --token YOUR_PAT
```

Your token is stored in the OS keychain -- never written to disk.

### 2. Verify connectivity

```bash
jira ping
```

### 3. Set a default project (optional)

The context system lets you set defaults so you don't have to repeat flags:

```bash
jira context set --project PROJ
jira context set --board-id 42
```

Now commands like `jira issue list` automatically filter to project `PROJ`.

### 4. Start using it

```bash
# List issues in your project
jira issue list

# View a specific issue
jira issue view PROJ-123

# Create an issue
jira issue create -p PROJ -s "Fix login bug" -t Bug

# Search with JQL
jira search jql "project = PROJ AND status = 'In Progress'"

# Pipe to jq
jira issue list | jq '.[].key'
```

## Output Formats

| Flag                 | Description                              |
|----------------------|------------------------------------------|
| `--format json`      | Machine-readable JSON (default)          |
| `--format markdown`  | Structured markdown optimized for LLMs   |

Set a persistent default with:

```bash
jira context set --display markdown
```

The `--format` flag always takes precedence over the context default.

## Writing to Jira

Reads are only half the job -- an agent has to *act*. Every mutation is a single command with explicit flags, so the line in the transcript is exactly what changed (see [Why a CLI instead of an MCP server](#5-why-a-cli-instead-of-an-mcp-server)). Like the read paths, write commands print the result as JSON and exit non-zero on failure, so an agent can chain them with `&&` and check success from the exit code.

```bash
# Create an issue (returns the new key on stdout)
jira issue create -p PROJ -s "Fix login bug" -t Bug -b "Steps to reproduce..."

# Edit fields, including custom fields by id (-F is repeatable)
jira issue edit PROJ-123 -s "New summary" -l backend -F customfield_10145="value"

# Transition through the workflow
jira issue move PROJ-123 "In Progress"
jira issue move PROJ-123 Done --resolution Fixed --comment "Shipped in v1.2"

# Assign and comment ('me' for yourself, 'x' to unassign)
jira issue assign PROJ-123 alice@example.com
jira issue comment add PROJ-123 "Investigated -- root cause was a stale cache."

# Link, clone, delete
jira issue link PROJ-123 PROJ-456 "blocks"
jira issue clone PROJ-123 -s "Follow-up: ..."
jira issue delete PROJ-123

# Sprint management
jira sprint add 42 PROJ-123 PROJ-456
```

Capture a created key and act on it in the same script:

```bash
key=$(jira issue create -p PROJ -s "Automated task" -t Task --raw | jq -r '.key')
jira issue move "$key" "In Progress" && jira issue assign "$key" me
```

Run `jira issue <verb> --help` for the full flag set on any write command.

## Commands

| Command         | Description                                  |
|-----------------|----------------------------------------------|
| `jira auth`     | Login, logout, status, whoami                |
| `jira config`   | Manage profiles (init, list, show, set, use, delete) |
| `jira context`  | Set default filters (project, board, labels, etc.)   |
| `jira issue`    | Full issue lifecycle (list, view, create, edit, delete, assign, move, comment, link, clone) |
| `jira board`    | List boards, view board issues               |
| `jira sprint`   | List, start, close sprints; add issues       |
| `jira project`  | List and view projects                       |
| `jira search`   | JQL and full-text search                     |
| `jira user`     | Search and get users                         |
| `jira me`       | Show current user                            |
| `jira open`     | Open project or issue in browser             |
| `jira ping`     | Check connectivity to Jira                   |

Run `jira <command> --help` for details on any command.

## Configuration

Config lives at `~/.config/jira-cli/config.yml`. You normally don't need to edit it by hand -- use the `jira config` and `jira context` commands instead.

### Profiles

Profiles let you manage multiple Jira instances:

```bash
# Create a profile for a second instance
jira config init --profile work --base-url https://work.atlassian.net
jira auth login --profile work --server https://work.atlassian.net \
  --email you@work.com --token YOUR_TOKEN

# Switch default profile
jira config use work

# Use a profile for a single command
jira issue list --profile work
```

### Environment Variables

These override config file values and are useful in CI/automation:

| Variable          | Description                         |
|-------------------|-------------------------------------|
| `JIRA_BASE_URL`   | Jira server URL                    |
| `JIRA_TOKEN`      | API token (bypasses OS keychain)   |
| `JIRA_EMAIL`      | User email                         |
| `JIRA_AUTH_TYPE`   | Auth type: `basic` or `pat`       |
| `JIRABOT_PROFILE`  | Profile name to use               |
