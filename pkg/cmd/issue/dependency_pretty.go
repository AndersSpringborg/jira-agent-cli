package issue

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

type prettyLayout struct {
	components []prettyComponent
}

type prettyComponent struct {
	layers [][]prettyLayoutNode
	edges  []prettyLayoutEdge
}

type prettyLayoutNode struct {
	id      string
	label   string
	virtual bool
}

type prettyLayoutEdge struct {
	from string
	to   string
}

type prettyGroup struct {
	id      string
	members []string
	label   string
}

func buildPrettyLayout(graph *dependencyGraph) prettyLayout {
	ready := make(map[string]bool, len(graph.Ready))
	for _, key := range graph.Ready {
		ready[key] = true
	}
	nodes := make(map[string]*dependencyNode, len(graph.Nodes))
	for index := range graph.Nodes {
		nodes[graph.Nodes[index].Key] = &graph.Nodes[index]
	}

	groups, groupByKey := prettyGroups(graph, nodes, ready)
	edgeSet := make(map[string]prettyLayoutEdge)
	neighbors := make(map[string][]string)
	for _, edge := range graph.Edges {
		from, fromOK := groupByKey[edge.From]
		to, toOK := groupByKey[edge.To]
		if !fromOK || !toOK || from == to {
			continue
		}
		key := from + "\x00" + to
		edgeSet[key] = prettyLayoutEdge{from: from, to: to}
	}
	for _, edge := range edgeSet {
		neighbors[edge.from] = append(neighbors[edge.from], edge.to)
		neighbors[edge.to] = append(neighbors[edge.to], edge.from)
	}
	for key := range neighbors {
		sort.Strings(neighbors[key])
	}

	groupByID := make(map[string]prettyGroup, len(groups))
	for _, group := range groups {
		groupByID[group.id] = group
	}
	seen := make(map[string]bool, len(groups))
	layout := prettyLayout{components: make([]prettyComponent, 0)}
	for _, group := range groups {
		if seen[group.id] {
			continue
		}
		ids := prettyConnectedGroups(group.id, neighbors, seen)
		componentEdges := make([]prettyLayoutEdge, 0)
		inComponent := make(map[string]bool, len(ids))
		for _, id := range ids {
			inComponent[id] = true
		}
		for _, edge := range edgeSet {
			if inComponent[edge.from] && inComponent[edge.to] {
				componentEdges = append(componentEdges, edge)
			}
		}
		sort.Slice(componentEdges, func(i, j int) bool {
			if componentEdges[i].from != componentEdges[j].from {
				return componentEdges[i].from < componentEdges[j].from
			}
			return componentEdges[i].to < componentEdges[j].to
		})
		layout.components = append(layout.components, buildPrettyComponent(ids, groupByID, componentEdges))
	}
	return layout
}

func prettyGroups(graph *dependencyGraph, nodes map[string]*dependencyNode, ready map[string]bool) ([]prettyGroup, map[string]string) {
	components := prettyStronglyConnectedComponents(graph.Nodes, graph.Edges)
	groups := make([]prettyGroup, 0, len(components))
	groupByKey := make(map[string]string, len(graph.Nodes))
	selfEdges := make(map[string]bool)
	for _, edge := range graph.Edges {
		if edge.From == edge.To {
			selfEdges[edge.From] = true
		}
	}
	for _, members := range components {
		id := members[0]
		cyclic := len(members) > 1 || selfEdges[id]
		labels := make([]string, 0, len(members))
		for _, key := range members {
			node := nodes[key]
			labels = append(labels, prettyNodeMarker(node, ready[key])+" "+key)
			groupByKey[key] = id
		}
		label := labels[0]
		if cyclic {
			label = prettyCycleLabel(members, labels, graph.Edges)
		}
		groups = append(groups, prettyGroup{id: id, members: members, label: label})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].id < groups[j].id })
	return groups, groupByKey
}

func prettyCycleLabel(members, labels []string, edges []dependencyEdge) string {
	memberSet := make(map[string]bool, len(members))
	labelByKey := make(map[string]string, len(members))
	for index, key := range members {
		memberSet[key] = true
		labelByKey[key] = labels[index]
	}
	internal := make([]dependencyEdge, 0)
	for _, edge := range edges {
		if memberSet[edge.From] && memberSet[edge.To] {
			internal = append(internal, edge)
		}
	}
	if len(members) == 2 && len(internal) == 2 &&
		internal[0].From == internal[1].To && internal[0].To == internal[1].From {
		return "↻ {" + strings.Join(labels, " ⇄ ") + "}"
	}
	relations := make([]string, 0, len(internal))
	for _, edge := range internal {
		relations = append(relations, labelByKey[edge.From]+" → "+labelByKey[edge.To])
	}
	return "↻ {" + strings.Join(relations, ", ") + "}"
}

func prettyNodeMarker(node *dependencyNode, ready bool) string {
	marker := "○"
	switch {
	case node.Resolved:
		marker = "✓"
	case !node.InScope:
		marker = "◇"
	case ready:
		marker = "●"
	}
	if node.Resolved && !node.InScope {
		marker = "✓◇"
	}
	return marker
}

func prettyStronglyConnectedComponents(nodes []dependencyNode, edges []dependencyEdge) [][]string {
	adjacency := make(map[string][]string)
	for _, edge := range edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}
	for key := range adjacency {
		sort.Strings(adjacency[key])
	}
	index := 0
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	onStack := make(map[string]bool)
	stack := make([]string, 0, len(nodes))
	components := make([][]string, 0, len(nodes))
	var visit func(string)
	visit = func(key string) {
		indices[key] = index
		lowlink[key] = index
		index++
		stack = append(stack, key)
		onStack[key] = true
		for _, next := range adjacency[key] {
			if _, exists := indices[next]; !exists {
				visit(next)
				lowlink[key] = min(lowlink[key], lowlink[next])
			} else if onStack[next] {
				lowlink[key] = min(lowlink[key], indices[next])
			}
		}
		if lowlink[key] != indices[key] {
			return
		}
		component := make([]string, 0, 1)
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
		components = append(components, component)
	}
	for index := range nodes {
		if _, exists := indices[nodes[index].Key]; !exists {
			visit(nodes[index].Key)
		}
	}
	sort.Slice(components, func(i, j int) bool { return components[i][0] < components[j][0] })
	return components
}

func prettyConnectedGroups(start string, neighbors map[string][]string, seen map[string]bool) []string {
	queue := []string{start}
	seen[start] = true
	ids := make([]string, 0, 1)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ids = append(ids, id)
		for _, neighbor := range neighbors[id] {
			if !seen[neighbor] {
				seen[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func buildPrettyComponent(ids []string, groups map[string]prettyGroup, edges []prettyLayoutEdge) prettyComponent {
	indegree := make(map[string]int, len(ids))
	outgoing := make(map[string][]string)
	rank := make(map[string]int, len(ids))
	for _, id := range ids {
		indegree[id] = 0
	}
	for _, edge := range edges {
		indegree[edge.to]++
		outgoing[edge.from] = append(outgoing[edge.from], edge.to)
	}
	for id := range outgoing {
		sort.Strings(outgoing[id])
	}
	queue := make([]string, 0)
	for _, id := range ids {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, target := range outgoing[id] {
			rank[target] = max(rank[target], rank[id]+1)
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, target)
				sort.Strings(queue)
			}
		}
	}

	maxRank := 0
	for _, id := range ids {
		maxRank = max(maxRank, rank[id])
	}
	layers := make([][]prettyLayoutNode, maxRank+1)
	for _, id := range ids {
		layers[rank[id]] = append(layers[rank[id]], prettyLayoutNode{id: id, label: groups[id].label})
	}
	segments := make([]prettyLayoutEdge, 0, len(edges))
	for _, edge := range edges {
		previous := edge.from
		for level := rank[edge.from] + 1; level < rank[edge.to]; level++ {
			virtualID := fmt.Sprintf("%s→%s@%d", edge.from, edge.to, level)
			layers[level] = append(layers[level], prettyLayoutNode{id: virtualID, label: "│", virtual: true})
			segments = append(segments, prettyLayoutEdge{from: previous, to: virtualID})
			previous = virtualID
		}
		segments = append(segments, prettyLayoutEdge{from: previous, to: edge.to})
	}
	for level := range layers {
		sort.Slice(layers[level], func(i, j int) bool { return layers[level][i].id < layers[level][j].id })
	}
	return prettyComponent{layers: layers, edges: segments}
}

func renderDependencyGraphPretty(graph *dependencyGraph) string {
	layout := buildPrettyLayout(graph)
	var text strings.Builder
	text.WriteString("Dependency graph (blocker ──▶ blocked)\n")
	text.WriteString("Legend: ● ready  ○ blocked  ✓ resolved  ◇ external  ↻ cycle\n")
	if len(layout.components) == 0 {
		text.WriteString("\nNo issues.\n")
		return text.String()
	}
	for index, component := range layout.components {
		fmt.Fprintf(&text, "\nComponent %d\n", index+1)
		text.WriteString(renderPrettyComponent(component))
	}
	return text.String()
}

func renderPrettyComponent(component prettyComponent) string {
	maxNodes := 1
	cellWidth := 0
	for _, layer := range component.layers {
		maxNodes = max(maxNodes, len(layer))
		for _, node := range layer {
			cellWidth = max(cellWidth, utf8.RuneCountInString(node.label)+6)
		}
	}
	width := maxNodes * cellWidth
	if maxNodes == 1 {
		width = cellWidth - 6
	}
	positions := make(map[string]int)
	virtual := make(map[string]bool)
	centersByLayer := make([][]int, len(component.layers))
	for level, layer := range component.layers {
		centersByLayer[level] = prettyLayerCenters(width, cellWidth, len(layer))
		for index, node := range layer {
			positions[node.id] = centersByLayer[level][index]
			virtual[node.id] = node.virtual
		}
	}

	var text strings.Builder
	for level, layer := range component.layers {
		labels := prettyBlankLine(width)
		for index, node := range layer {
			placePrettyText(labels, centersByLayer[level][index], node.label)
		}
		text.WriteString(strings.TrimRight(string(labels), " "))
		text.WriteByte('\n')
		if level == len(component.layers)-1 {
			continue
		}
		edges := make([]prettyLayoutEdge, 0)
		fromLayer := make(map[string]bool, len(layer))
		toLayer := make(map[string]bool, len(component.layers[level+1]))
		for _, node := range layer {
			fromLayer[node.id] = true
		}
		for _, node := range component.layers[level+1] {
			toLayer[node.id] = true
		}
		for _, edge := range component.edges {
			if fromLayer[edge.from] && toLayer[edge.to] {
				edges = append(edges, edge)
			}
		}
		text.WriteString(renderPrettyRoutes(width, edges, positions, virtual))
	}
	return text.String()
}

func prettyLayerCenters(width, cellWidth, count int) []int {
	centers := make([]int, count)
	start := (width - (count-1)*cellWidth) / 2
	for index := range centers {
		centers[index] = start + index*cellWidth
	}
	return centers
}

func prettyBlankLine(width int) []rune {
	line := make([]rune, width)
	for index := range line {
		line[index] = ' '
	}
	return line
}

func placePrettyText(line []rune, center int, value string) {
	runes := []rune(value)
	start := center - len(runes)/2
	for index, char := range runes {
		if start+index >= 0 && start+index < len(line) {
			line[start+index] = char
		}
	}
}

const (
	prettyUp = 1 << iota
	prettyDown
	prettyLeft
	prettyRight
)

func renderPrettyRoutes(width int, edges []prettyLayoutEdge, positions map[string]int, virtual map[string]bool) string {
	if len(edges) == 0 {
		return ""
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from != edges[j].from {
			return edges[i].from < edges[j].from
		}
		return edges[i].to < edges[j].to
	})
	straight := true
	for _, edge := range edges {
		if positions[edge.from] != positions[edge.to] {
			straight = false
			break
		}
	}
	if straight {
		line := prettyBlankLine(width)
		for _, edge := range edges {
			line[positions[edge.from]] = '│'
		}
		var text strings.Builder
		text.WriteString(strings.TrimRight(string(line), " "))
		text.WriteByte('\n')
		for _, edge := range edges {
			if virtual[edge.to] {
				line[positions[edge.to]] = '│'
			} else {
				line[positions[edge.to]] = '▼'
			}
		}
		text.WriteString(strings.TrimRight(string(line), " "))
		text.WriteByte('\n')
		return text.String()
	}

	lanes := prettyRouteLanes(edges)
	height := 3
	for _, lane := range lanes {
		height = max(height, lane+2)
	}
	grid := make([][]int, height)
	for row := range grid {
		grid[row] = make([]int, width)
	}
	for index, edge := range edges {
		fromX := positions[edge.from]
		toX := positions[edge.to]
		lane := lanes[index]
		prettyVertical(grid, fromX, 0, lane)
		prettyHorizontal(grid, lane, fromX, toX)
		prettyVertical(grid, toX, lane, height-1)
	}
	var text strings.Builder
	targets := make(map[int]bool)
	virtualTargets := make(map[int]bool)
	for _, edge := range edges {
		x := positions[edge.to]
		targets[x] = true
		virtualTargets[x] = virtual[edge.to]
	}
	for row := range grid {
		line := make([]rune, width)
		for column, mask := range grid[row] {
			line[column] = prettyConnector(mask)
		}
		if row == height-1 {
			for column := range targets {
				if virtualTargets[column] {
					line[column] = '│'
				} else {
					line[column] = '▼'
				}
			}
		}
		text.WriteString(strings.TrimRight(string(line), " "))
		text.WriteByte('\n')
	}
	return text.String()
}

func prettyRouteLanes(edges []prettyLayoutEdge) []int {
	sources := make(map[string]bool)
	targets := make(map[string]bool)
	for _, edge := range edges {
		sources[edge.from] = true
		targets[edge.to] = true
	}
	lanes := make([]int, len(edges))
	if len(sources) == 1 || len(targets) == 1 {
		for index := range lanes {
			lanes[index] = 1
		}
		return lanes
	}
	laneBySource := make(map[string]int)
	nextLane := 1
	for index, edge := range edges {
		lane, exists := laneBySource[edge.from]
		if !exists {
			lane = nextLane
			nextLane++
			laneBySource[edge.from] = lane
		}
		lanes[index] = lane
	}
	return lanes
}

func prettyVertical(grid [][]int, column, fromRow, toRow int) {
	if fromRow > toRow {
		fromRow, toRow = toRow, fromRow
	}
	for row := fromRow; row < toRow; row++ {
		grid[row][column] |= prettyDown
		grid[row+1][column] |= prettyUp
	}
}

func prettyHorizontal(grid [][]int, row, fromColumn, toColumn int) {
	if fromColumn > toColumn {
		fromColumn, toColumn = toColumn, fromColumn
	}
	for column := fromColumn; column < toColumn; column++ {
		grid[row][column] |= prettyRight
		grid[row][column+1] |= prettyLeft
	}
}

func prettyConnector(mask int) rune {
	switch mask {
	case prettyUp, prettyDown, prettyUp | prettyDown:
		return '│'
	case prettyLeft, prettyRight, prettyLeft | prettyRight:
		return '─'
	case prettyDown | prettyRight:
		return '┌'
	case prettyDown | prettyLeft:
		return '┐'
	case prettyUp | prettyRight:
		return '└'
	case prettyUp | prettyLeft:
		return '┘'
	case prettyUp | prettyDown | prettyRight:
		return '├'
	case prettyUp | prettyDown | prettyLeft:
		return '┤'
	case prettyLeft | prettyRight | prettyDown:
		return '┬'
	case prettyLeft | prettyRight | prettyUp:
		return '┴'
	case prettyUp | prettyDown | prettyLeft | prettyRight:
		return '┼'
	default:
		return ' '
	}
}

func writeDependencyGraphPretty(writer io.Writer, graph *dependencyGraph) error {
	_, err := io.WriteString(writer, renderDependencyGraphPretty(graph))
	return err
}
