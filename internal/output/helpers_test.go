package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendColumns(t *testing.T) {
	t.Run("appends new columns", func(t *testing.T) {
		got := AppendColumns([]string{"key", "summary"}, []string{"customfield_10145"})
		assert.Equal(t, []string{"key", "summary", "customfield_10145"}, got)
	})

	t.Run("skips columns already present", func(t *testing.T) {
		got := AppendColumns([]string{"key", "customfield_10145"}, []string{"customfield_10145"})
		assert.Equal(t, []string{"key", "customfield_10145"}, got)
	})

	t.Run("no extra columns leaves cols unchanged", func(t *testing.T) {
		got := AppendColumns([]string{"key"}, nil)
		assert.Equal(t, []string{"key"}, got)
	})
}
