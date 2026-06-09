package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMyChanges(t *testing.T) {
	day := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)

	issues := []any{
		map[string]any{
			"key":    "TEST-1",
			"fields": map[string]any{"summary": "Login bug"},
			"changelog": map[string]any{
				"histories": []any{
					// Mine, on the target day -> kept (two items).
					map[string]any{
						"created": "2026-06-08T10:30:00.000+0000",
						"author":  map[string]any{"accountId": "me-123"},
						"items": []any{
							map[string]any{"field": "status", "fromString": "To Do", "toString": "In Progress"},
							map[string]any{"field": "assignee", "fromString": "", "toString": "Me"},
						},
					},
					// Mine, but a different day -> dropped.
					map[string]any{
						"created": "2026-06-07T09:00:00.000+0000",
						"author":  map[string]any{"accountId": "me-123"},
						"items":   []any{map[string]any{"field": "priority", "fromString": "Low", "toString": "High"}},
					},
					// Someone else, target day -> dropped.
					map[string]any{
						"created": "2026-06-08T12:00:00.000+0000",
						"author":  map[string]any{"accountId": "other-999"},
						"items":   []any{map[string]any{"field": "status", "fromString": "In Progress", "toString": "Done"}},
					},
				},
			},
		},
	}

	rows := collectMyChanges(issues, "me-123", day)

	require.Len(t, rows, 2)
	assert.Equal(t, "TEST-1", rows[0]["key"])
	assert.Equal(t, "Login bug", rows[0]["summary"])
	assert.Equal(t, "status", rows[0]["field"])
	assert.Equal(t, "To Do", rows[0]["from"])
	assert.Equal(t, "In Progress", rows[0]["to"])
	assert.Equal(t, "assignee", rows[1]["field"])
}

func TestCollectMyChanges_MatchesByName(t *testing.T) {
	// Server/DC identifies the author by "name" rather than accountId.
	day := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	issues := []any{
		map[string]any{
			"key": "DC-1",
			"changelog": map[string]any{
				"histories": []any{
					map[string]any{
						"created": "2026-06-08T08:00:00.000+0000",
						"author":  map[string]any{"name": "jsmith"},
						"items":   []any{map[string]any{"field": "summary", "fromString": "a", "toString": "b"}},
					},
				},
			},
		},
	}

	rows := collectMyChanges(issues, "jsmith", day)
	require.Len(t, rows, 1)
	assert.Equal(t, "summary", rows[0]["field"])
}
