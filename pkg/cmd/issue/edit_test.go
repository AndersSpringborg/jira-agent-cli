package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCustomFields(t *testing.T) {
	t.Run("plain values stay strings", func(t *testing.T) {
		got, err := parseCustomFields([]string{"customfield_10001=5", "customfield_10002=Option A"})
		require.NoError(t, err)
		assert.Equal(t, "5", got["customfield_10001"])
		assert.Equal(t, "Option A", got["customfield_10002"])
	})

	t.Run("JSON object value is parsed for user-picker fields", func(t *testing.T) {
		got, err := parseCustomFields([]string{`customfield_10145={"accountId":"712020:abc"}`})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"accountId": "712020:abc"}, got["customfield_10145"])
	})

	t.Run("JSON array value is parsed for multi-value fields", func(t *testing.T) {
		got, err := parseCustomFields([]string{`customfield_10010=[{"value":"A"},{"value":"B"}]`})
		require.NoError(t, err)
		assert.Equal(t, []any{
			map[string]any{"value": "A"},
			map[string]any{"value": "B"},
		}, got["customfield_10010"])
	})

	t.Run("malformed JSON object returns an error", func(t *testing.T) {
		_, err := parseCustomFields([]string{`customfield_10145={"accountId":}`})
		require.Error(t, err)
	})

	t.Run("missing = returns an error", func(t *testing.T) {
		_, err := parseCustomFields([]string{"customfield_10001"})
		require.Error(t, err)
	})
}
