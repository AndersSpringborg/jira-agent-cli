# Authentication & profiles

**The user authenticates themselves — the agent never handles the token.** Give the user the commands to run and let them run them (in Claude Code they can run a line inline with `! jira auth login ...`; otherwise they paste it into their own terminal). Never ask the user to paste a token into the conversation, and never put a real token on a command line you execute.

The auth type is auto-detected from the base URL: `*.atlassian.net` -> `basic` (email + API token), anything else -> `pat` (Personal Access Token). Tokens live in the OS keychain - never on disk.

Cloud profiles use REST v3 + ADF bodies. Jira Server / Data Center profiles use REST v2 + wiki/plain text bodies.

## Jira Cloud (`*.atlassian.net`)

1. User creates an API token at https://id.atlassian.com/manage-profile/security/api-tokens.
2. Initialize the profile and log in:

```bash
jira config init --base-url https://your-org.atlassian.net
jira auth login \
  --server https://your-org.atlassian.net \
  --email USER_EMAIL \
  --token API_TOKEN
```

## Jira Server / Data Center (PAT)

1. User creates a Personal Access Token under Profile -> Personal Access Tokens.
2. Initialize and log in:

```bash
jira config init --base-url https://jira.example.com
jira auth login \
  --server https://jira.example.com \
  --token PERSONAL_ACCESS_TOKEN
```

## Verifying

| Command              | What it checks                                  |
|----------------------|-------------------------------------------------|
| `jira auth status`   | Local keychain only - token present?            |
| `jira ping`          | Round-trips the API - token + network working   |
| `jira auth whoami`   | Like ping, but returns the authenticated user   |

## Multiple profiles

```bash
# Create a second profile
jira config init --profile work --base-url https://work.atlassian.net
jira auth login --profile work --server https://work.atlassian.net --email you@work.com --token TOKEN

# Switch the default
jira config use --profile work

# One-off with a different profile
jira issue list --profile personal
```

## Environment variables (CI / automation)

Set these to bypass interactive setup and the OS keychain.

| Variable           | Description                              |
|--------------------|------------------------------------------|
| `JIRA_BASE_URL`    | Jira server URL                          |
| `JIRA_TOKEN`       | API token or PAT                         |
| `JIRA_EMAIL`       | User email (Cloud / basic auth only)     |
| `JIRA_AUTH_TYPE`   | `basic` or `pat` (overrides detection)   |
| `JIRABOT_PROFILE`  | Profile name to use                      |
