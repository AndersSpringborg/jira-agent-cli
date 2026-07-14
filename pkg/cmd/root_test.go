package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutePrintsFailingCommandHelpOnError(t *testing.T) {
	root := NewRootCmd("test", "today")
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{"issue", "view"})

	err := Execute(root)
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "accepts 1 arg(s), received 0")
	assert.Contains(t, stderr.String(), "Usage:")
	assert.Contains(t, stderr.String(), "jira issue view <issue-key>")
	assert.Contains(t, stderr.String(), "--comments")
}
