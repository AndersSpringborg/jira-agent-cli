package issue

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUserSearcher struct {
	users []map[string]any
	err   error
}

func (s stubUserSearcher) ListUsers(string) ([]map[string]any, error) {
	return s.users, s.err
}

func TestResolveCommentBody(t *testing.T) {
	t.Run("uses positional body before template", func(t *testing.T) {
		got, err := resolveCommentBody([]string{"inline"}, "missing.md")
		require.NoError(t, err)
		assert.Equal(t, "inline", got)
	})

	t.Run("loads template", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "comment.md")
		require.NoError(t, os.WriteFile(path, []byte("\nfrom template\n"), 0o600))
		got, err := resolveCommentBody(nil, path)
		require.NoError(t, err)
		assert.Equal(t, "from template", got)
	})

	t.Run("requires body or template", func(t *testing.T) {
		_, err := resolveCommentBody(nil, "")
		require.Error(t, err)
	})
}

func TestParseViewFields(t *testing.T) {
	got := parseViewFields("summary, status", []string{"customfield_1", " labels "})
	assert.Equal(t, []string{"summary", "status", "customfield_1", "labels"}, got)
}

func TestResolveAssignmentUser(t *testing.T) {
	searcher := stubUserSearcher{users: []map[string]any{
		{"accountId": "wrong", "emailAddress": "other@example.com", "name": "other"},
		{"accountId": "abc123", "emailAddress": "alice@example.com", "name": "alice"},
	}}
	accountID, name := resolveAssignmentUser(searcher, "alice@example.com")
	assert.Equal(t, "abc123", accountID)
	assert.Equal(t, "alice", name)
}
