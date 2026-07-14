package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusJQLSupportsCommaSeparatedStatuses(t *testing.T) {
	assert.Equal(t, `status = "To Do"`, statusJQL("To Do"))
	assert.Equal(t, `status in ("define", "to do", "backlog")`, statusJQL("define, to do,backlog"))
	assert.Equal(t, "", statusJQL(" , "))
}

func TestListDefaultsToFiftyIssues(t *testing.T) {
	cmd := newListCmd(nil)

	maxResults, err := cmd.Flags().GetInt("max")
	require.NoError(t, err)
	assert.Equal(t, 50, maxResults)
}
