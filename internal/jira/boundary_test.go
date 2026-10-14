package jira_test

import (
	"testing"

	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/jira"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safeJQL = `project in ("CER") AND labels = "safe"`

func boundedClient(t *testing.T) (*fakeJira, *jira.Client) {
	t.Helper()
	fake := newFakeJira(t)
	client := fake.client(t)
	client.SetBoundary(&config.Boundary{Projects: []string{"CER"}, Labels: []string{"safe"}})
	return fake, client
}

func TestSearchWithoutBoundaryIsUnchanged(t *testing.T) {
	fake := newFakeJira(t)
	client := fake.client(t)

	data, err := client.Search("project = CER", 0, 50)

	require.NoError(t, err)
	assert.Equal(t, []string{"project = CER"}, fake.jql)
	assert.Len(t, issueKeys(data), 4)
}

func TestSearchIsConfinedToBoundaryAndKeepsOrdering(t *testing.T) {
	fake, client := boundedClient(t)

	data, err := client.Search("project = OPS ORDER BY updated DESC", 0, 50)

	require.NoError(t, err)
	assert.Equal(t, "("+safeJQL+") AND (project = OPS) ORDER BY updated DESC", fake.jql[0])
	assert.Equal(t, []string{"CER-1", "CER-3"}, issueKeys(data), "issues Jira returns outside the boundary are dropped")
}

func TestSearchWithOnlyOrderingUsesBoundaryAlone(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.Search("order by created", 0, 50)

	require.NoError(t, err)
	assert.Equal(t, safeJQL+" order by created", fake.jql[0])
}

func TestSearchDoesNotSplitOrderByInsideQuotes(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.Search(`summary ~ "order by"`, 0, 50)

	require.NoError(t, err)
	assert.Equal(t, "("+safeJQL+`) AND (summary ~ "order by")`, fake.jql[0])
}

func TestSearchRejectsJQLThatBreaksOutOfTheBoundary(t *testing.T) {
	fake, client := boundedClient(t)

	for _, jql := range []string{`x) OR (project = OPS`, `summary ~ "unterminated`, `labels = safe)`} {
		_, err := client.Search(jql, 0, 50)
		require.ErrorIs(t, err, jira.ErrOutsideBoundary, jql)
	}
	assert.Empty(t, fake.jql)
}

func TestSearchRedactsLinkedIssuesOutsideBoundary(t *testing.T) {
	_, client := boundedClient(t)

	data, err := client.Search("", 0, 50)

	require.NoError(t, err)
	first := data["issues"].([]any)[0].(map[string]any)
	assertRedacted(t, first)
}

func TestGetIssueOutsideBoundaryFailsWithoutFetchingIt(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.GetIssue("CER-2", nil)

	require.ErrorIs(t, err, jira.ErrOutsideBoundary)
	assert.Contains(t, err.Error(), "CER-2")
	assert.Equal(t, []string{"(" + safeJQL + ") AND (issuekey in (CER-2))"}, fake.jql)
}

func TestGetIssueRejectsMalformedKeys(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.GetIssue("CER-1) OR (project = OPS", nil)

	require.Error(t, err)
	assert.Empty(t, fake.jql)
}

func TestGetIssueInsideBoundaryRedactsEmbeddedIssues(t *testing.T) {
	_, client := boundedClient(t)

	data, err := client.GetIssue("cer-1", nil)

	require.NoError(t, err)
	assertRedacted(t, data)
}

func assertRedacted(t *testing.T, issue map[string]any) {
	t.Helper()
	fields := issue["fields"].(map[string]any)
	links := fields["issuelinks"].([]any)
	require.Len(t, links, 1)
	assert.Equal(t, "CER-3", links[0].(map[string]any)["outwardIssue"].(map[string]any)["key"])
	subtasks := fields["subtasks"].([]any)
	require.Len(t, subtasks, 1)
	assert.Equal(t, "CER-3", subtasks[0].(map[string]any)["key"])
	assert.NotContains(t, fields, "parent", "parent OPS-1 is outside the project boundary")
}

func TestMutationsOutsideBoundaryNeverReachJira(t *testing.T) {
	fake, client := boundedClient(t)

	calls := map[string]func() error{
		"edit":       func() error { return client.EditIssue("CER-2", map[string]any{"summary": "x"}, nil) },
		"delete":     func() error { return client.DeleteIssue("CER-2") },
		"assign":     func() error { return client.AssignIssue("CER-2", "", "bob", "") },
		"transition": func() error { return client.TransitionIssue("CER-2", "31") },
		"comment":    func() error { return client.AddComment("CER-2", "hi") },
		"link":       func() error { return client.LinkIssues("CER-1", "CER-2", "Blocks") },
		"unlink":     func() error { return client.DeleteIssueLink("101") },
		"sprint":     func() error { return client.MoveIssuesToSprint(7, []string{"CER-1", "CER-2"}) },
		"clone": func() error {
			_, err := client.CloneIssue("CER-2", nil)
			return err
		},
		"transitions": func() error {
			_, err := client.GetIssueTransitions("CER-2")
			return err
		},
	}
	for name, call := range calls {
		require.ErrorIs(t, call(), jira.ErrOutsideBoundary, name)
	}
	assert.Empty(t, fake.writes)
}

func TestMutationsInsideBoundaryReachJira(t *testing.T) {
	fake, client := boundedClient(t)

	require.NoError(t, client.AddComment("CER-1", "hi"))
	require.NoError(t, client.LinkIssues("CER-1", "CER-3", "Blocks"))
	require.NoError(t, client.DeleteIssueLink("100"))

	assert.Equal(t, []string{
		"POST /rest/api/2/issue/CER-1/comment",
		"POST /rest/api/2/issueLink",
		"DELETE /rest/api/2/issueLink/100",
	}, fake.writes)
}

func TestEditCannotRemoveBoundaryLabels(t *testing.T) {
	fake, client := boundedClient(t)

	removeErr := client.EditIssue("CER-1", nil, map[string]any{"labels": []any{map[string]any{"remove": "safe"}}})
	setErr := client.EditIssue("CER-1", map[string]any{"labels": []string{"other"}}, nil)
	transitionErr := client.TransitionIssueWithUpdates("CER-1", "31", map[string]any{"labels": []any{}}, nil)

	require.ErrorIs(t, removeErr, jira.ErrOutsideBoundary)
	require.ErrorIs(t, setErr, jira.ErrOutsideBoundary)
	require.ErrorIs(t, transitionErr, jira.ErrOutsideBoundary)
	assert.Empty(t, fake.writes)

	require.NoError(t, client.EditIssue("CER-1", nil, map[string]any{"labels": []any{map[string]any{"add": "x"}}}))
	assert.Len(t, fake.writes, 1)
}

func TestCreateIssueStaysInsideBoundary(t *testing.T) {
	fake, client := boundedClient(t)

	_, outsideErr := client.CreateIssue("OPS", "s", "Task", "", "", nil, "", nil, nil)
	_, parentErr := client.CreateIssue("CER", "s", "Sub-task", "", "", nil, "CER-2", nil, nil)
	require.ErrorIs(t, outsideErr, jira.ErrOutsideBoundary)
	require.ErrorIs(t, parentErr, jira.ErrOutsideBoundary)
	assert.Empty(t, fake.writes)

	_, err := client.CreateIssue("CER", "s", "Task", "", "", []string{"backend"}, "", nil, nil)
	require.NoError(t, err)
	fields := fake.bodies[0]["fields"].(map[string]any)
	assert.ElementsMatch(t, []any{"backend", "safe"}, fields["labels"])
}

func TestCloneCannotLeaveBoundary(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.CloneIssue("CER-1", map[string]any{"project": map[string]any{"key": "OPS"}})
	require.ErrorIs(t, err, jira.ErrOutsideBoundary)

	_, err = client.CloneIssue("CER-1", map[string]any{"labels": []string{"other"}})
	require.NoError(t, err)
	fields := fake.bodies[0]["fields"].(map[string]any)
	assert.ElementsMatch(t, []any{"other", "safe"}, fields["labels"])
}

func TestSprintIssuesAreConfinedToBoundary(t *testing.T) {
	fake, client := boundedClient(t)

	data, err := client.GetSprintIssues(7)

	require.NoError(t, err)
	assert.Equal(t, safeJQL, fake.jql[0])
	assert.Equal(t, []string{"CER-1", "CER-3"}, issueKeys(data))
}

func TestProjectsOutsideBoundaryAreHidden(t *testing.T) {
	_, client := boundedClient(t)

	projects, err := client.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "CER", projects[0]["key"])

	_, err = client.GetProject("OPS")
	require.ErrorIs(t, err, jira.ErrOutsideBoundary)
}

func TestRawRequestsAreRefusedInsideBoundary(t *testing.T) {
	fake, client := boundedClient(t)

	_, err := client.Raw("GET", "/rest/api/2/issue/CER-2", nil, nil)

	require.ErrorIs(t, err, jira.ErrOutsideBoundary)
	assert.Contains(t, err.Error(), "sudo")
	assert.Empty(t, fake.jql)
}
