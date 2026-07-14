# Lessons

- Status filters are user-facing list inputs, not atomic values. When adding commands that mirror issue-list filters, support comma-separated statuses consistently across both existing and new issue commands (for example `--status "define,to do,backlog"`) and test whitespace/empty segments.
- A dependency visualization must render the graph topology itself, not an adjacency list. Build a complete layout model first, then draw continuous branch and join lines so shared downstream steps are visually connected rather than repeated as references.
