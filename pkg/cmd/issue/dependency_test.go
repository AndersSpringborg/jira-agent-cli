package issue

import (
	"bytes"
	"encoding/json"
	"testing"

	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDependencyGraph(t *testing.T) {
	issues := []map[string]any{
		mockIssue("PROJ-1", "Foundation", nil, []any{
			mockLink("Blocks", "PROJ-1", "PROJ-2"),
		}),
		mockIssue("PROJ-2", "Feature", nil, []any{
			mockLink("Blocks", "PROJ-1", "PROJ-2"),
		}),
		mockIssue("PROJ-3", "Resolved prerequisite", map[string]any{"name": "Done"}, []any{
			mockLink("Blocks", "PROJ-3", "PROJ-4"),
		}),
		mockIssue("PROJ-4", "Ready after resolution", nil, []any{
			mockLink("Blocks", "PROJ-3", "PROJ-4"),
		}),
	}

	graph, external := buildDependencyGraph(issues, "Blocks")
	require.Empty(t, external)
	assert.Equal(t, []string{"PROJ-1", "PROJ-4"}, graph.Ready)
	assert.Equal(t, []dependencyEdge{
		{From: "PROJ-1", To: "PROJ-2", Type: "Blocks"},
		{From: "PROJ-3", To: "PROJ-4", Type: "Blocks"},
	}, graph.Edges)
	require.Len(t, graph.Blocked, 1)
	assert.Equal(t, "PROJ-2", graph.Blocked[0].Key)
	assert.Equal(t, []string{"PROJ-1"}, graph.Blocked[0].BlockedBy)
}

func TestBuildDependencyGraphUnderstandsJiraLinkDirection(t *testing.T) {
	issues := []map[string]any{
		mockIssue("PROJ-1", "Blocker", nil, []any{
			mockPartialLink("Blocks", "inwardIssue", "PROJ-2"),
		}),
		mockIssue("PROJ-2", "Blocked", nil, []any{
			mockPartialLink("Blocks", "outwardIssue", "PROJ-1"),
		}),
	}

	graph, external := buildDependencyGraph(issues, "Blocks")
	require.Empty(t, external)
	assert.Equal(t, []dependencyEdge{{From: "PROJ-1", To: "PROJ-2", Type: "Blocks"}}, graph.Edges)
	assert.Equal(t, []string{"PROJ-1"}, graph.Ready)
}

func TestBuildDependencyGraphRetainsExternalBlockers(t *testing.T) {
	issues := []map[string]any{
		mockIssue("PROJ-1", "Blocked feature", nil, []any{
			mockLink("Blocks", "OTHER-9", "PROJ-1"),
		}),
	}

	graph, external := buildDependencyGraph(issues, "blocks")
	assert.Equal(t, []string{"OTHER-9"}, external)
	assert.Empty(t, graph.Ready)
	require.Len(t, graph.Nodes, 2)
	assert.False(t, graph.Nodes[0].InScope)
	assert.Equal(t, "OTHER-9", graph.Nodes[0].Key)
}

func TestExternalResolvedBlockerMakesScopedIssueReady(t *testing.T) {
	graph, external := buildDependencyGraph([]map[string]any{
		mockIssue("PROJ-1", "Feature", nil, []any{mockLink("Blocks", "OTHER-9", "PROJ-1")}),
	}, "Blocks")
	require.Equal(t, []string{"OTHER-9"}, external)

	graph.updateExternal(mockIssue("OTHER-9", "Finished prerequisite", map[string]any{"name": "Done"}, nil))
	assert.Equal(t, []string{"PROJ-1"}, graph.Ready)
	assert.Empty(t, graph.Nodes[0].BlockedBy)
}

func TestEmptyDependencyGraphUsesJSONArrays(t *testing.T) {
	graph, _ := buildDependencyGraph(nil, "Blocks")
	encoded, err := json.Marshal(graph)
	require.NoError(t, err)
	assert.JSONEq(t, `{"nodes":[],"edges":[],"ready":[],"blocked":[],"cycles":[]}`, string(encoded))
}

func TestBuildDependencyGraphFindsCycles(t *testing.T) {
	issues := []map[string]any{
		mockIssue("PROJ-1", "One", nil, []any{mockLink("Blocks", "PROJ-1", "PROJ-2")}),
		mockIssue("PROJ-2", "Two", nil, []any{mockLink("Blocks", "PROJ-2", "PROJ-1")}),
	}

	graph, _ := buildDependencyGraph(issues, "Blocks")
	assert.Equal(t, [][]string{{"PROJ-1", "PROJ-2"}}, graph.Cycles)
	assert.Empty(t, graph.Ready)
}

func TestReadyNodesPrioritizeIssuesThatUnblockMoreWork(t *testing.T) {
	issues := []map[string]any{
		mockIssue("PROJ-1", "High leverage", nil, []any{
			mockLink("Blocks", "PROJ-1", "PROJ-3"),
			mockLink("Blocks", "PROJ-1", "PROJ-4"),
		}),
		mockIssue("PROJ-2", "Independent", nil, nil),
		mockIssue("PROJ-3", "Three", nil, []any{mockLink("Blocks", "PROJ-1", "PROJ-3")}),
		mockIssue("PROJ-4", "Four", nil, []any{mockLink("Blocks", "PROJ-1", "PROJ-4")}),
	}

	graph, _ := buildDependencyGraph(issues, "Blocks")
	ready := graph.readyNodes()
	require.Len(t, ready, 2)
	assert.Equal(t, "PROJ-1", ready[0].Key)
	assert.Equal(t, []string{"PROJ-3", "PROJ-4"}, ready[0].Unblocks)
	assert.Equal(t, "PROJ-2", ready[1].Key)
}

func TestDependencyGraphMarkdownIsStructured(t *testing.T) {
	graph, _ := buildDependencyGraph([]map[string]any{
		mockIssue("PROJ-1", "Foundation", nil, []any{mockLink("Blocks", "PROJ-1", "PROJ-2")}),
		mockIssue("PROJ-2", "Feature", nil, []any{mockLink("Blocks", "PROJ-1", "PROJ-2")}),
	}, "Blocks")
	var rendered bytes.Buffer
	driver := output.NewDriverWithWriter(output.FormatMarkdown, &rendered)

	require.NoError(t, writeDependencyGraphMarkdown(driver, graph))
	assert.Contains(t, rendered.String(), "**Ready:** PROJ-1")
	assert.Contains(t, rendered.String(), "| Blocker | Blocked | Type |")
	assert.Contains(t, rendered.String(), "| PROJ-1 | PROJ-2 | Blocks |")
}

func TestDependencyOptionsUseContextFilters(t *testing.T) {
	opts := &dependencyOptions{}
	jql, err := opts.jql(&config.Context{
		Project: "PROJ", Status: "To Do", Labels: []string{"backend"}, Epic: "PROJ-9",
	})
	require.NoError(t, err)
	assert.Equal(t, `project = PROJ AND ("Epic Link" = "PROJ-9" OR parent = "PROJ-9") AND status = "To Do" AND labels in ("backend") ORDER BY updated DESC`, jql)
}

func TestDependencyCommandsAreRegisteredWithLLMHelp(t *testing.T) {
	cmd := NewCmd(nil)
	ready, _, err := cmd.Find([]string{"ready"})
	require.NoError(t, err)
	assert.Contains(t, ready.Long, "resolution field")
	assert.NotNil(t, ready.Flags().Lookup("link-type"))

	graph, _, err := cmd.Find([]string{"graph"})
	require.NoError(t, err)
	assert.Contains(t, graph.Long, "blocker to blocked")
}

func mockIssue(key, summary string, resolution any, links []any) map[string]any {
	return map[string]any{
		"key": key,
		"fields": map[string]any{
			"summary":    summary,
			"status":     map[string]any{"name": "To Do"},
			"resolution": resolution,
			"issuelinks": links,
		},
	}
}

func mockPartialLink(linkType, direction, key string) map[string]any {
	return map[string]any{
		"type":    map[string]any{"name": linkType},
		direction: map[string]any{"key": key},
	}
}

func mockLink(linkType, blocker, blocked string) map[string]any {
	return map[string]any{
		"type":         map[string]any{"name": linkType},
		"outwardIssue": map[string]any{"key": blocker},
		"inwardIssue":  map[string]any{"key": blocked},
	}
}
