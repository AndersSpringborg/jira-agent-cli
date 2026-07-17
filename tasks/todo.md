# Issue dependency commands

## graph-pretty full-layout correction
- [x] Replace adjacency snapshots with failing top-down full-graph snapshots, including visible branch joins.
- [x] Build a deterministic layered layout model from the complete graph before rendering.
- [x] Render continuous top-down branch/join connectors, disconnected components, external/resolved states, and cycle groups.
- [x] Update mocked output in README, skill docs, task review, and the open PR.
- [x] Run formatting, focused/full tests, build, lint, and diff checks; commit and push only intended files.

## graph-pretty implementation plan

- [x] Add failing snapshot/string tests for chains, branches/shared dependencies, disconnected nodes, resolved/external nodes, and cycles.
- [x] Register `jira issue graph-pretty` with reused scope/link flags, strong long help, and examples.
- [x] Implement deterministic line rendering without recursive duplication or cycle loops.
- [x] Update README and Jira CLI skill with mocked `graph-pretty` output.
- [x] Run focused tests, formatting, full tests, build, lint, and `git diff --check`.
- [x] Review the diff, update this review section and PR #7 description, stage only intended files, commit, and push.

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

## graph-pretty review

- Added a deterministic per-component adjacency view: each node has one full labeled row and sorted outgoing blocker-to-blocked connectors, avoiding recursive subtree duplication and cycle traversal.
- State labels and symbols distinguish ready, blocked, resolved, external/out-of-scope, and cycle nodes while retaining summaries and disconnected components.
- Snapshot tests cover a chain, branch with shared dependency, disconnected node, resolved external blocker, and cycle; command tests cover registration, help semantics, and inherited link flags.
- Verification passed: `go test ./...`, `go build -o bin/jira ./cmd/jira`, `golangci-lint run ./...` (0 issues), and `git diff --check`.
- Fresh-context reviewer subagents were attempted, but the local async runner exited before producing results; parent-side review found no additional changes required.
- PR #7 was already merged before the feature commits were pushed, so the implementation is proposed in follow-up PR #8: https://github.com/AndersSpringborg/jira-agent-cli/pull/8

## graph-pretty full-layout review

- Replaced the adjacency listing with a two-phase implementation: build a deterministic layout from the complete graph, then render it.
- The layout condenses strongly connected components, layers the resulting DAG by dependency depth, preserves long edges through virtual routing nodes, and separates weakly connected components.
- The renderer uses compact issue keys and continuous top-down Unicode routes. Branches fan out and shared downstream dependencies visibly join before their target.
- Cycle groups preserve their internal directed edges; ready, blocked, resolved, external, and cyclic states remain visible through compact markers.
- Snapshot tests cover chains, branch/join diamonds, disconnected components, resolved external blockers, and cycles. A layout-model test verifies the joined graph before rendering.
- Verification passed: `go test ./...`, `go build -o bin/jira ./cmd/jira`, `golangci-lint run ./...` (0 issues), and `git diff --check`.

# Named Jira contexts

- [x] Reproduce the profile/project persistence limitation in a command-level test.
- [x] Add named contexts that persist project filters and authentication profile together.
- [x] Store contexts as a global YAML list with explicit `name` fields.
- [x] Verify multiple contexts retain independent settings and can be switched.
- [x] Run tests, lint, formatting, and a manual smoke test.

## Named context review

```bash
jira context set cai --project CAI --profile trifork
jira context use cai
jira context list
```

Contexts are stored as a global YAML list with explicit `name`, `profile`, and filter fields. The active context supplies both its filters and authentication profile to subsequent commands. An explicit per-command `--profile` still overrides the context profile.

Validation passed: `go test ./...`, `golangci-lint run ./...` (0 issues), `git diff --check`, and a manual two-context set/use/show/list smoke test.
