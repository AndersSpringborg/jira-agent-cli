package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDefaultsToFiftyIssues(t *testing.T) {
	cmd := newListCmd(nil)

	maxResults, err := cmd.Flags().GetInt("max")
	require.NoError(t, err)
	assert.Equal(t, 50, maxResults)
}
