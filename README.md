# jira-cli

A non-interactive CLI for Jira designed for AI agents and automation. All output is machine-readable JSON by default, making it ideal for `jq` pipelines and LLM tool calling.

## Credit

This project is inspired by [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli) -- a feature-rich **interactive** Jira command line with a full TUI (tables, keyboard navigation, interactive prompts). Think of it as the [k9s](https://k9scli.io/) of Jira: powerful, visual, and built for humans at a terminal.

This project takes a different approach. It is the **kubectl** of Jira: non-interactive, scriptable, and designed for AI agents. No TUI, no prompts -- just structured output that machines can parse. If you want a great interactive experience, use [ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli). If you want to wire Jira into an AI agent, CI pipeline, or shell script, use this.

## Install

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
