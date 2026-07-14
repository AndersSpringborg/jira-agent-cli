package issue

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

type dependencyNode struct {
	Key       string   `json:"key"`
	Summary   string   `json:"summary,omitempty"`
	Status    string   `json:"status,omitempty"`
	Resolved  bool     `json:"resolved"`
	InScope   bool     `json:"inScope"`
	Unblocks  []string `json:"unblocks,omitempty"`
	BlockedBy []string `json:"blockedBy,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Priority  string   `json:"priority,omitempty"`
}

type dependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type blockedIssue struct {
	Key       string   `json:"key"`
	BlockedBy []string `json:"blockedBy"`
}

type dependencyGraph struct {
	Nodes   []dependencyNode `json:"nodes"`
	Edges   []dependencyEdge `json:"edges"`
	Ready   []string         `json:"ready"`
	Blocked []blockedIssue   `json:"blocked"`
	Cycles  [][]string       `json:"cycles"`
}

func buildDependencyGraph(issues []map[string]any, linkType string) (*dependencyGraph, []string) {
	nodes := make(map[string]dependencyNode, len(issues))
	for _, issue := range issues {
		node := dependencyNodeFromIssue(issue, true)
		if node.Key != "" {
			nodes[node.Key] = node
		}
	}

	edgeSet := make(map[string]dependencyEdge)
	for _, issue := range issues {
		currentKey, _ := issue["key"].(string)
		fields, _ := issue["fields"].(map[string]any)
		links, _ := fields["issuelinks"].([]any)
		for _, rawLink := range links {
			link, _ := rawLink.(map[string]any)
			typeData, _ := link["type"].(map[string]any)
			name, _ := typeData["name"].(string)
			if !strings.EqualFold(name, linkType) {
				continue
			}

			inward := linkedIssue(link["inwardIssue"])
			outward := linkedIssue(link["outwardIssue"])
			var blocker, blocked dependencyNode
			switch {
			case inward.Key != "" && outward.Key != "":
				blocker, blocked = outward, inward
			case outward.Key != "":
				blocker = outward
				blocked = nodes[currentKey]
			case inward.Key != "":
				blocker = nodes[currentKey]
				blocked = inward
			default:
				continue
			}
			if blocker.Key == "" || blocked.Key == "" {
				continue
			}
			if _, ok := nodes[blocker.Key]; !ok {
				nodes[blocker.Key] = blocker
			}
			if _, ok := nodes[blocked.Key]; !ok {
				nodes[blocked.Key] = blocked
			}
			edge := dependencyEdge{From: blocker.Key, To: blocked.Key, Type: name}
			edgeSet[edge.From+"\x00"+edge.To+"\x00"+strings.ToLower(edge.Type)] = edge
		}
	}

	graph := &dependencyGraph{
		Nodes:   make([]dependencyNode, 0, len(nodes)),
		Edges:   make([]dependencyEdge, 0, len(edgeSet)),
		Ready:   []string{},
		Blocked: []blockedIssue{},
		Cycles:  [][]string{},
	}
	for _, edge := range edgeSet {
		graph.Edges = append(graph.Edges, edge)
	}
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From != graph.Edges[j].From {
			return graph.Edges[i].From < graph.Edges[j].From
		}
		return graph.Edges[i].To < graph.Edges[j].To
	})
	for key := range nodes {
		graph.Nodes = append(graph.Nodes, nodes[key])
	}
	graph.analyze()

	var external []string
	for i := range graph.Nodes {
		if !graph.Nodes[i].InScope {
			external = append(external, graph.Nodes[i].Key)
		}
	}
	return graph, external
}

func linkedIssue(value any) dependencyNode {
	issue, _ := value.(map[string]any)
	return dependencyNodeFromIssue(issue, false)
}

func dependencyNodeFromIssue(issue map[string]any, inScope bool) dependencyNode {
	key, _ := issue["key"].(string)
	fields, _ := issue["fields"].(map[string]any)
	return dependencyNode{
		Key:      key,
		Summary:  dependencyStringField(fields, "summary"),
		Status:   nestedStringField(fields, "status", "name"),
		Resolved: fields != nil && fields["resolution"] != nil,
		InScope:  inScope,
		Assignee: nestedStringField(fields, "assignee", "displayName"),
		Priority: nestedStringField(fields, "priority", "name"),
	}
}

func dependencyStringField(fields map[string]any, key string) string {
	value, _ := fields[key].(string)
	return value
}

func nestedStringField(fields map[string]any, key, nestedKey string) string {
	nested, _ := fields[key].(map[string]any)
	value, _ := nested[nestedKey].(string)
	return value
}

func (g *dependencyGraph) updateExternal(issue map[string]any) {
	updated := dependencyNodeFromIssue(issue, false)
	for i := range g.Nodes {
		if g.Nodes[i].Key == updated.Key {
			g.Nodes[i] = updated
			break
		}
	}
	g.analyze()
}

func (g *dependencyGraph) analyze() {
	for i := range g.Nodes {
		g.Nodes[i].BlockedBy = nil
		g.Nodes[i].Unblocks = nil
	}
	nodeByKey := make(map[string]*dependencyNode, len(g.Nodes))
	for i := range g.Nodes {
		nodeByKey[g.Nodes[i].Key] = &g.Nodes[i]
	}
	for _, edge := range g.Edges {
		from, fromOK := nodeByKey[edge.From]
		to, toOK := nodeByKey[edge.To]
		if !fromOK || !toOK || from.Resolved {
			continue
		}
		to.BlockedBy = append(to.BlockedBy, edge.From)
		if !to.Resolved {
			from.Unblocks = append(from.Unblocks, edge.To)
		}
	}
	for i := range g.Nodes {
		sort.Strings(g.Nodes[i].BlockedBy)
		sort.Strings(g.Nodes[i].Unblocks)
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].Key < g.Nodes[j].Key })

	g.Ready = g.Ready[:0]
	g.Blocked = g.Blocked[:0]
	for i := range g.Nodes {
		node := &g.Nodes[i]
		if !node.InScope || node.Resolved {
			continue
		}
		if len(node.BlockedBy) == 0 {
			g.Ready = append(g.Ready, node.Key)
		} else {
			g.Blocked = append(g.Blocked, blockedIssue{Key: node.Key, BlockedBy: node.BlockedBy})
		}
	}
	g.Cycles = dependencyCycles(g.Nodes, g.Edges)
}

func (g *dependencyGraph) readyNodes() []dependencyNode {
	readySet := make(map[string]bool, len(g.Ready))
	for _, key := range g.Ready {
		readySet[key] = true
	}
	ready := make([]dependencyNode, 0, len(g.Ready))
	for i := range g.Nodes {
		if readySet[g.Nodes[i].Key] {
			ready = append(ready, g.Nodes[i])
		}
	}
	sort.Slice(ready, func(i, j int) bool {
		if len(ready[i].Unblocks) != len(ready[j].Unblocks) {
			return len(ready[i].Unblocks) > len(ready[j].Unblocks)
		}
		return ready[i].Key < ready[j].Key
	})
	return ready
}

func dependencyCycles(nodes []dependencyNode, edges []dependencyEdge) [][]string {
	unresolved := make(map[string]bool, len(nodes))
	for i := range nodes {
		unresolved[nodes[i].Key] = !nodes[i].Resolved
	}
	adjacency := make(map[string][]string)
	for _, edge := range edges {
		if unresolved[edge.From] && unresolved[edge.To] {
			adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		}
	}
	for key := range adjacency {
		sort.Strings(adjacency[key])
	}

	index := 0
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	cycles := make([][]string, 0)
	var visit func(string)
	visit = func(key string) {
		indices[key] = index
		lowlink[key] = index
		index++
		stack = append(stack, key)
		onStack[key] = true
		for _, next := range adjacency[key] {
			if _, seen := indices[next]; !seen {
				visit(next)
				if lowlink[next] < lowlink[key] {
					lowlink[key] = lowlink[next]
				}
			} else if onStack[next] && indices[next] < lowlink[key] {
				lowlink[key] = indices[next]
			}
		}
		if lowlink[key] != indices[key] {
			return
		}
		var component []string
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == key {
				break
			}
		}
		sort.Strings(component)
		if len(component) > 1 || (len(component) == 1 && contains(adjacency[component[0]], component[0])) {
			cycles = append(cycles, component)
		}
	}
	keys := make([]string, 0, len(unresolved))
	for key, isUnresolved := range unresolved {
		if isUnresolved {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, seen := indices[key]; !seen {
			visit(key)
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return strings.Join(cycles[i], "\x00") < strings.Join(cycles[j], "\x00") })
	return cycles
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type dependencyOptions struct {
	project    string
	assignee   string
	status     string
	issueType  string
	epic       string
	labels     []string
	maxResults int
	linkType   string
}

func (o *dependencyOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.project, "project", "p", "", "Project key")
	cmd.Flags().StringVar(&o.assignee, "assignee", "", "Filter by assignee (use 'currentUser()' for self)")
	cmd.Flags().StringVar(&o.status, "status", "", "Filter by status (comma-separated for multiple)")
	cmd.Flags().StringVarP(&o.issueType, "type", "t", "", "Filter by issue type")
	cmd.Flags().StringVar(&o.epic, "epic", "", "Filter by epic issue key")
	cmd.Flags().StringSliceVar(&o.labels, "label", nil, "Filter by label (repeatable)")
	cmd.Flags().IntVar(&o.maxResults, "max", 50, "Max scoped issues")
	cmd.Flags().StringVar(&o.linkType, "link-type", "Blocks", "Jira dependency link type")
}

func (o *dependencyOptions) jql(ctx *config.Context) (string, error) {
	if ctx != nil {
		if o.project == "" {
			o.project = ctx.Project
		}
		if o.assignee == "" {
			o.assignee = ctx.Assignee
		}
		if o.status == "" {
			o.status = ctx.Status
		}
		if o.issueType == "" {
			o.issueType = ctx.IssueType
		}
		if o.epic == "" {
			o.epic = ctx.Epic
		}
		if len(o.labels) == 0 {
			o.labels = ctx.Labels
		}
	}
	if o.project == "" {
		return "", fmt.Errorf("no project specified; use --project or set context with `jira context set --project PROJ`")
	}
	parts := []string{fmt.Sprintf("project = %s", o.project)}
	if o.epic != "" {
		parts = append(parts, fmt.Sprintf(`("Epic Link" = %q OR parent = %q)`, o.epic, o.epic))
	}
	if o.assignee != "" {
		parts = append(parts, fmt.Sprintf("assignee = %s", o.assignee))
	}
	if clause := statusJQL(o.status); clause != "" {
		parts = append(parts, clause)
	}
	if o.issueType != "" {
		parts = append(parts, fmt.Sprintf(`issuetype = %q`, o.issueType))
	}
	if len(o.labels) > 0 {
		quoted := make([]string, len(o.labels))
		for i, label := range o.labels {
			quoted[i] = fmt.Sprintf("%q", label)
		}
		parts = append(parts, fmt.Sprintf("labels in (%s)", strings.Join(quoted, ", ")))
	}
	return strings.Join(parts, " AND ") + " ORDER BY updated DESC", nil
}

func loadDependencyGraph(f *cmdutil.Factory, opts *dependencyOptions) (*dependencyGraph, error) {
	profile, err := f.LoadProfile()
	if err != nil {
		return nil, err
	}
	jql, err := opts.jql(profile.Context)
	if err != nil {
		return nil, err
	}
	client, err := f.LoadClient()
	if err != nil {
		return nil, err
	}
	data, err := client.Search(jql, 0, opts.maxResults, "issuelinks")
	if err != nil {
		return nil, err
	}
	rawIssues, _ := data["issues"].([]any)
	issues := make([]map[string]any, 0, len(rawIssues))
	for _, raw := range rawIssues {
		if issue, ok := raw.(map[string]any); ok {
			issues = append(issues, issue)
		}
	}
	graph, external := buildDependencyGraph(issues, opts.linkType)
	for _, key := range external {
		issue, getErr := client.GetIssue(key, []string{"summary", "status", "resolution", "assignee", "priority"})
		if getErr != nil {
			return nil, fmt.Errorf("load external dependency %s: %w", key, getErr)
		}
		graph.updateExternal(issue)
	}
	return graph, nil
}

func writeDependencyGraphMarkdown(driver output.DisplayDriver, graph *dependencyGraph) error {
	var text strings.Builder
	text.WriteString("## Dependency graph\n\n")
	ready := strings.Join(graph.Ready, ", ")
	if ready == "" {
		ready = "None"
	}
	fmt.Fprintf(&text, "**Ready:** %s\n\n", ready)
	text.WriteString("### Nodes\n\n")
	text.WriteString("| Key | Summary | Status | Resolved | In scope | Blocked by | Unblocks |\n")
	text.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		fmt.Fprintf(&text, "| %s | %s | %s | %t | %t | %s | %s |\n",
			markdownCell(node.Key), markdownCell(node.Summary), markdownCell(node.Status),
			node.Resolved, node.InScope, markdownCell(strings.Join(node.BlockedBy, ", ")),
			markdownCell(strings.Join(node.Unblocks, ", ")))
	}
	text.WriteString("\n### Edges (blocker to blocked)\n\n")
	text.WriteString("| Blocker | Blocked | Type |\n| --- | --- | --- |\n")
	for _, edge := range graph.Edges {
		fmt.Fprintf(&text, "| %s | %s | %s |\n", markdownCell(edge.From), markdownCell(edge.To), markdownCell(edge.Type))
	}
	text.WriteString("\n### Cycles\n\n")
	if len(graph.Cycles) == 0 {
		text.WriteString("None.\n")
	} else {
		for _, cycle := range graph.Cycles {
			fmt.Fprintf(&text, "- %s\n", markdownCell(strings.Join(cycle, " -> ")))
		}
	}
	return driver.Message("%s", strings.TrimSuffix(text.String(), "\n"))
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", `\|`)
	return strings.ReplaceAll(value, "\n", " ")
}

func renderDependencyGraphPretty(graph *dependencyGraph) string {
	var text strings.Builder
	text.WriteString("Dependency graph (blocker ──▶ blocked)\n")
	text.WriteString("Legend: ● ready  ○ blocked  ✓ resolved  ◇ external  ↻ cycle\n")
	if len(graph.Nodes) == 0 {
		text.WriteString("\nNo issues.\n")
		return text.String()
	}

	outgoing := make(map[string][]string)
	neighbors := make(map[string][]string)
	for _, edge := range graph.Edges {
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
		neighbors[edge.From] = append(neighbors[edge.From], edge.To)
		neighbors[edge.To] = append(neighbors[edge.To], edge.From)
	}
	for key := range outgoing {
		sort.Strings(outgoing[key])
	}
	for key := range neighbors {
		sort.Strings(neighbors[key])
	}

	cycleNodes := make(map[string]bool)
	for _, cycle := range graph.Cycles {
		for _, key := range cycle {
			cycleNodes[key] = true
		}
	}
	readyNodes := make(map[string]bool, len(graph.Ready))
	for _, key := range graph.Ready {
		readyNodes[key] = true
	}

	seen := make(map[string]bool, len(graph.Nodes))
	componentNumber := 0
	for i := range graph.Nodes {
		if seen[graph.Nodes[i].Key] {
			continue
		}
		componentNumber++
		component := dependencyComponent(graph.Nodes[i].Key, neighbors, seen)
		fmt.Fprintf(&text, "\nComponent %d\n", componentNumber)
		for _, key := range component {
			node := dependencyNodeByKey(graph.Nodes, key)
			marker, states := prettyDependencyState(&node, readyNodes[key], cycleNodes[key])
			fmt.Fprintf(&text, "%s %s [%s]", marker, node.Key, strings.Join(states, ", "))
			if summary := strings.Join(strings.Fields(node.Summary), " "); summary != "" {
				fmt.Fprintf(&text, " %s", summary)
			}
			text.WriteByte('\n')
			for edgeIndex, target := range outgoing[key] {
				connector := "├──▶"
				if edgeIndex == len(outgoing[key])-1 {
					connector = "└──▶"
				}
				fmt.Fprintf(&text, "%s %s\n", connector, target)
			}
		}
	}
	return text.String()
}

func dependencyComponent(start string, neighbors map[string][]string, seen map[string]bool) []string {
	queue := []string{start}
	seen[start] = true
	component := make([]string, 0, 1)
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		component = append(component, key)
		for _, neighbor := range neighbors[key] {
			if !seen[neighbor] {
				seen[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	sort.Strings(component)
	return component
}

func dependencyNodeByKey(nodes []dependencyNode, key string) dependencyNode {
	index := sort.Search(len(nodes), func(i int) bool { return nodes[i].Key >= key })
	if index < len(nodes) && nodes[index].Key == key {
		return nodes[index]
	}
	return dependencyNode{Key: key}
}

func prettyDependencyState(node *dependencyNode, ready, cycle bool) (string, []string) {
	states := make([]string, 0, 3)
	marker := "○"
	switch {
	case node.Resolved:
		marker = "✓"
		states = append(states, "resolved")
	case cycle:
		marker = "↻"
		states = append(states, "blocked")
	case !node.InScope:
		marker = "◇"
	case ready:
		marker = "●"
		states = append(states, "ready")
	default:
		states = append(states, "blocked")
	}
	if !node.InScope {
		states = append(states, "external")
	}
	if cycle {
		states = append(states, "cycle")
	}
	return marker, states
}

func writeDependencyGraphPretty(writer io.Writer, graph *dependencyGraph) error {
	_, err := io.WriteString(writer, renderDependencyGraphPretty(graph))
	return err
}

func newReadyCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &dependencyOptions{}
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "List unresolved issues with no unresolved blockers",
		Long: `List actionable issues in the active project/context dependency graph.

An issue is ready when it has no unresolved incoming Blocks links. A linked
blocker is resolved only when its Jira resolution field is set. Results are
ordered by how many unresolved issues they directly unblock, then by key.

Examples:
  jira issue ready
  jira issue ready --project PROJ
  jira issue ready --label backend --format markdown
  jira issue ready --link-type Depends`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			graph, err := loadDependencyGraph(f, opts)
			if err != nil {
				return err
			}
			readyNodes := graph.readyNodes()
			rows := make([]map[string]any, 0, len(readyNodes))
			for i := range readyNodes {
				node := &readyNodes[i]
				rows = append(rows, map[string]any{
					"key": node.Key, "summary": node.Summary, "status": node.Status,
					"assignee": node.Assignee, "priority": node.Priority,
					"unblocks": node.Unblocks, "unblocksCount": len(node.Unblocks),
				})
			}
			return f.DisplayDriver(cmd).List("Ready issues", []string{"key", "summary", "status", "assignee", "priority", "unblocksCount", "unblocks"}, rows)
		},
	}
	opts.bindFlags(cmd)
	return cmd
}

func newGraphPrettyCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &dependencyOptions{}
	cmd := &cobra.Command{
		Use:   "graph-pretty",
		Short: "Draw a human-readable issue dependency graph",
		Long: `Draw the active project/context dependency graph with Unicode connecting lines.

Each component lists every issue once, followed by its blocker-to-blocked edges.
This adjacency layout keeps branches and shared dependencies explicit without
recursively duplicating subtrees or looping forever on cycles. Markers and labels
identify ready, blocked, resolved, external/out-of-scope, and cyclic issues.
A blocker is resolved only when its Jira resolution field is set.

The command uses the same project/context scope, filters, external dependency
hydration, and --link-type behavior as issue graph. Output is always plain text,
regardless of the global --format setting, for direct terminal inspection.

Examples:
  jira issue graph-pretty
  jira issue graph-pretty --project PROJ
  jira issue graph-pretty --status "Define,To Do,Backlog"
  jira issue graph-pretty --label backend --link-type Depends`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			graph, err := loadDependencyGraph(f, opts)
			if err != nil {
				return err
			}
			return writeDependencyGraphPretty(cmd.OutOrStdout(), graph)
		},
	}
	opts.bindFlags(cmd)
	return cmd
}

func newGraphCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &dependencyOptions{}
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Display the issue dependency graph",
		Long: `Display a directed dependency graph for the active project/context.

Edges point from blocker to blocked issue. The stable JSON object contains
nodes, edges, ready issue keys, blocked issues with their blockers, and cycles.
Linked issues outside the selected scope are included with inScope=false.

Examples:
  jira issue graph
  jira issue graph --project PROJ | jq '.ready'
  jira issue graph --format markdown
  jira issue graph --link-type Depends`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			graph, err := loadDependencyGraph(f, opts)
			if err != nil {
				return err
			}
			driver := f.DisplayDriver(cmd)
			if _, markdown := driver.(*output.MarkdownDriver); markdown {
				return writeDependencyGraphMarkdown(driver, graph)
			}
			return driver.Raw(graph)
		},
	}
	opts.bindFlags(cmd)
	return cmd
}
