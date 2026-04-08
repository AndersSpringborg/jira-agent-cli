package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"AndersSpringborg/jira-cli/internal/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- JSON Driver Tests ---

func TestJSONDriver_Item(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	data := map[string]any{
		"key": "PROJ-123",
		"fields": map[string]any{
			"summary": "Fix bug",
			"status":  map[string]any{"name": "In Progress"},
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "PROJ-123", result["key"])
	fields := result["fields"].(map[string]any)
	assert.Equal(t, "Fix bug", fields["summary"])
}

func TestJSONDriver_List(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	rows := []map[string]any{
		{"id": float64(1), "name": "Board A", "type": "scrum"},
		{"id": float64(2), "name": "Board B", "type": "kanban"},
	}

	err := d.List("Boards", []string{"id", "name", "type"}, rows)
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, "Board A", result[0]["name"])
	assert.Equal(t, "Board B", result[1]["name"])
}

func TestJSONDriver_Raw(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	data := map[string]any{"total": float64(42), "issues": []any{}}

	err := d.Raw(data)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, float64(42), result["total"])
}

func TestJSONDriver_Message(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	err := d.Message("Created issue: %s", "PROJ-456")
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Created issue: PROJ-456", result["message"])
}

func TestJSONDriver_Error(t *testing.T) {
	// Error() writes to os.Stderr, not the driver's writer.
	// We verify it doesn't return an error.
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	err := d.Error(assert.AnError)
	require.NoError(t, err)

	// The buffer should be empty since Error() writes to stderr.
	assert.Empty(t, buf.String())
}

// --- Markdown Driver Tests ---

func TestMarkdownDriver_Item_Issue(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"key": "PROJ-123",
		"fields": map[string]any{
			"summary":     "Fix login redirect bug",
			"status":      map[string]any{"name": "In Progress"},
			"assignee":    map[string]any{"displayName": "Jane Smith"},
			"priority":    map[string]any{"name": "High"},
			"issuetype":   map[string]any{"name": "Bug"},
			"reporter":    map[string]any{"displayName": "John Doe"},
			"created":     "2024-01-15T10:30:00.000+0000",
			"updated":     "2024-01-16T14:20:00.000+0000",
			"description": "When a user tries to log in with SSO, they are redirected to a blank page.",
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	out := buf.String()

	// Should have a heading with key and summary
	assert.Contains(t, out, "## PROJ-123: Fix login redirect bug")

	// Should have a field table
	assert.Contains(t, out, "| Field | Value |")
	assert.Contains(t, out, "| Key | PROJ-123 |")
	assert.Contains(t, out, "| Status | In Progress |")
	assert.Contains(t, out, "| Assignee | Jane Smith |")
	assert.Contains(t, out, "| Priority | High |")
	assert.Contains(t, out, "| Type | Bug |")
	assert.Contains(t, out, "| Reporter | John Doe |")

	// Should have description section
	assert.Contains(t, out, "### Description")
	assert.Contains(t, out, "redirected to a blank page")
}

func TestMarkdownDriver_Item_WithComments(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"key": "PROJ-123",
		"fields": map[string]any{
			"summary": "Test issue",
			"comment": map[string]any{
				"comments": []any{
					map[string]any{
						"author":  map[string]any{"displayName": "Alice"},
						"body":    "I can reproduce this",
						"created": "2024-01-15",
					},
					map[string]any{
						"author":  map[string]any{"displayName": "Bob"},
						"body":    "Working on it",
						"created": "2024-01-16",
					},
				},
			},
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "### Comments (2)")
	assert.Contains(t, out, "**Alice**")
	assert.Contains(t, out, "I can reproduce this")
	assert.Contains(t, out, "**Bob**")
	assert.Contains(t, out, "Working on it")
}

func TestMarkdownDriver_Item_SimpleObject(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"id":    float64(42),
		"name":  "Sprint 1",
		"state": "active",
	}

	err := d.Item("Sprint", data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "## Sprint")
	assert.Contains(t, out, "| Name | Sprint 1 |")
	assert.Contains(t, out, "| State | active |")
}

func TestMarkdownDriver_List(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	rows := []map[string]any{
		{"key": "PROJ-1", "summary": "First issue", "status": map[string]any{"name": "Done"}},
		{"key": "PROJ-2", "summary": "Second issue", "status": map[string]any{"name": "To Do"}},
	}

	err := d.List("Issues", []string{"key", "summary", "status"}, rows)
	require.NoError(t, err)

	out := buf.String()

	assert.Contains(t, out, "## Issues (2)")
	assert.Contains(t, out, "| key | summary | status |")
	assert.Contains(t, out, "| --- | --- | --- |")
	assert.Contains(t, out, "| PROJ-1 | First issue | Done |")
	assert.Contains(t, out, "| PROJ-2 | Second issue | To Do |")
}

func TestMarkdownDriver_List_Empty(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	err := d.List("Issues", []string{"key"}, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No issues found.")
}

func TestMarkdownDriver_Raw_FallsBackToJSON(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{"total": float64(1)}

	err := d.Raw(data)
	require.NoError(t, err)

	// Should produce valid JSON even from markdown driver
	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result["total"])
}

func TestMarkdownDriver_Message(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	err := d.Message("Created issue: %s", "PROJ-789")
	require.NoError(t, err)
	assert.Equal(t, "Created issue: PROJ-789\n", buf.String())
}

func TestMarkdownDriver_Error(t *testing.T) {
	// Error() writes to os.Stderr, not the driver's writer.
	// We verify it doesn't return an error.
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	err := d.Error(assert.AnError)
	require.NoError(t, err)

	// The buffer should be empty since Error() writes to stderr.
	assert.Empty(t, buf.String())
}

// --- Helper Function Tests ---

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"int-like float", float64(42), "42"},
		{"float", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"map with name", map[string]any{"name": "High"}, "High"},
		{"map with displayName", map[string]any{"displayName": "Jane"}, "Jane"},
		{"map with key", map[string]any{"key": "PROJ-1"}, "PROJ-1"},
		{"empty map", map[string]any{}, ""},
		{"string slice", []string{"a", "b"}, "a, b"},
		{"any slice", []any{"x", "y"}, "x, y"},
		{"nested map slice", []any{
			map[string]any{"name": "A"},
			map[string]any{"name": "B"},
		}, "A, B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := output.FormatValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeFields(t *testing.T) {
	defaults := []string{"key", "summary", "status"}

	// Empty string returns defaults
	assert.Equal(t, defaults, output.NormalizeFields("", defaults))

	// User-specified columns
	assert.Equal(t, []string{"key", "assignee"}, output.NormalizeFields("key,assignee", defaults))

	// Trims whitespace
	assert.Equal(t, []string{"key", "summary"}, output.NormalizeFields(" key , summary ", defaults))

	// Ignores empty parts
	result := output.NormalizeFields("key,,summary", defaults)
	assert.Equal(t, []string{"key", "summary"}, result)
}

func TestParseFormat(t *testing.T) {
	f, err := output.ParseFormat("json")
	assert.NoError(t, err)
	assert.Equal(t, output.FormatJSON, f)

	f, err = output.ParseFormat("")
	assert.NoError(t, err)
	assert.Equal(t, output.FormatJSON, f)

	f, err = output.ParseFormat("markdown")
	assert.NoError(t, err)
	assert.Equal(t, output.FormatMarkdown, f)

	f, err = output.ParseFormat("md")
	assert.NoError(t, err)
	assert.Equal(t, output.FormatMarkdown, f)

	_, err = output.ParseFormat("xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

// --- Interface Compliance Tests ---

func TestJSONDriverImplementsDisplayDriver(t *testing.T) {
	var _ output.DisplayDriver = output.NewDriverWithWriter(output.FormatJSON, &bytes.Buffer{}) //nolint:staticcheck // interface compliance check
}

func TestMarkdownDriverImplementsDisplayDriver(t *testing.T) {
	var _ output.DisplayDriver = output.NewDriverWithWriter(output.FormatMarkdown, &bytes.Buffer{}) //nolint:staticcheck // interface compliance check
}

// --- Integration-style tests: verify real Jira-like data flows ---

func TestJSONDriver_RealIssueViewFlow(t *testing.T) {
	// Simulate what `issue view PROJ-123` would produce
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatJSON, &buf)

	apiResponse := map[string]any{
		"key": "PROJ-123",
		"fields": map[string]any{
			"summary":  "Fix login redirect bug",
			"status":   map[string]any{"name": "In Progress"},
			"assignee": map[string]any{"displayName": "Jane Smith"},
		},
	}

	err := d.Item("Issue", apiResponse)
	require.NoError(t, err)

	// An AI agent should be able to parse this with jq:
	// jq '.key' -> "PROJ-123"
	// jq '.fields.status.name' -> "In Progress"
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "PROJ-123", parsed["key"])
}

func TestMarkdownDriver_RealSearchFlow(t *testing.T) {
	// Simulate what `search jql "project=PROJ"` would produce
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	rows := []map[string]any{
		{
			"key":      "PROJ-1",
			"summary":  "First issue",
			"status":   map[string]any{"name": "Done"},
			"assignee": map[string]any{"displayName": "Alice"},
			"priority": map[string]any{"name": "High"},
		},
		{
			"key":      "PROJ-2",
			"summary":  "Second issue",
			"status":   map[string]any{"name": "To Do"},
			"assignee": nil,
			"priority": map[string]any{"name": "Low"},
		},
	}

	err := d.List("Issues", []string{"key", "summary", "status", "assignee", "priority"}, rows)
	require.NoError(t, err)

	out := buf.String()

	// LLM should see a clean markdown table
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.GreaterOrEqual(t, len(lines), 4) // header + separator + 2 data rows

	// Nil assignee should render as empty string
	assert.Contains(t, out, "PROJ-2")
}
