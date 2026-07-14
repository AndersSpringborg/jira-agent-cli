# Issue dependency commands

## Agreed CLI

- `jira issue ready`: list unresolved issues in the active project/context that have no unresolved `Blocks` dependencies.
- `jira issue graph`: emit the dependency graph for the active project/context.
- A blocker is resolved only when Jira's `resolution` field is set.
- Every failed command prints the error and the failing command's full help text to stderr.

## Proposed output contract

- JSON graph output is a stable object with `nodes`, directed `edges` (`from` blocker, `to` blocked), `ready`, `blocked`, and `cycles`.
- Nodes identify scope membership and resolution state; external linked blockers are retained so the graph is not misleading.
- Markdown graph output uses compact sections/tables rather than a visual-only format, keeping it readable by LLMs.
- `ready` returns list rows with issue identity/status plus `unblocks` keys/count for dependency-resolution prioritization.
- Both commands accept the same project/context filters as `issue list`, plus `--max`; `--link-type` defaults to `Blocks` for Jira instances with custom dependency link types.

## Implementation plan

- [x] Add failing tests for deterministic graph construction, resolved blocker handling, external blockers, cycles, and ready ordering.
- [x] Add failing command tests for registration/default flags and full help on errors.
- [x] Reuse the issue-list context/filter contract for ready and graph scope JQL.
- [x] Implement dependency graph parsing and analysis as pure functions.
- [x] Implement `jira issue ready` and `jira issue graph` with JSON/markdown output.
- [x] Make command failures print error plus command-specific help to stderr.
- [x] Update README and the Jira CLI skill with examples/output semantics.
- [x] Run focused tests, `go test ./...`, formatting, build, and lint.
- [x] Support comma-separated `--status` filters consistently in issue list, ready, and graph.
- [x] Review the final diff and record verification results here.

## Review

- Added deterministic graph analysis with blocker-to-blocked edges, resolution-aware readiness, external dependency hydration, strongly connected cycle detection, and empty-array JSON contracts.
- `issue ready` sorts actionable work by direct unresolved dependents, then issue key. `issue graph` emits stable JSON and structured markdown tables.
- Failure handling is centralized in `cmd.Execute`; the original error and full failing-command help are written to stderr.
- Tests cover Jira's one-sided issue-link representations, resolved and external blockers, cycles, ordering, context JQL, comma-separated status filters, command registration/help, markdown rendering, JSON empty collections, and command failure help.
- Verification passed: `go test ./...`, `go build -o bin/jira ./cmd/jira`, `golangci-lint run ./...` (0 issues), and `git diff --check`.
- Manual failure check: `./bin/jira issue view` printed the argument error followed by complete `jira issue view` help.
- Reviewer subagents were attempted but the local subagent runner repeatedly exited/timed out without a review result; a parent-side diff review found no additional changes required.
