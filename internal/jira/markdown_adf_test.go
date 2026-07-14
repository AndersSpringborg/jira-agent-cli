package jira

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestMarkdownToADF(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string yields empty doc",
			in:   "",
			want: `{"content":[],"type":"doc","version":1}`,
		},
		{
			name: "plain prose becomes a paragraph",
			in:   "Just text.",
			want: `{"content":[{"content":[{"text":"Just text.","type":"text"}],"type":"paragraph"}],"type":"doc","version":1}`,
		},
		{
			name: "heading carries its level",
			in:   "## Problem",
			want: `{"content":[{"attrs":{"level":2},"content":[{"text":"Problem","type":"text"}],"type":"heading"}],"type":"doc","version":1}`,
		},
		{
			name: "fenced code block keeps language and content",
			in:   "```text\n9 pools * 100\n```",
			want: `{"content":[{"attrs":{"language":"text"},"content":[{"text":"9 pools * 100","type":"text"}],"type":"codeBlock"}],"type":"doc","version":1}`,
		},
		{
			name: "bullet list becomes bulletList with listItems",
			in:   "- one\n- two",
			want: `{"content":[{"content":[{"content":[{"content":[{"text":"one","type":"text"}],"type":"paragraph"}],"type":"listItem"},{"content":[{"content":[{"text":"two","type":"text"}],"type":"paragraph"}],"type":"listItem"}],"type":"bulletList"}],"type":"doc","version":1}`,
		},
		{
			name: "ordered list becomes orderedList",
			in:   "1. first\n2. second",
			want: `{"content":[{"content":[{"content":[{"content":[{"text":"first","type":"text"}],"type":"paragraph"}],"type":"listItem"},{"content":[{"content":[{"text":"second","type":"text"}],"type":"paragraph"}],"type":"listItem"}],"type":"orderedList"}],"type":"doc","version":1}`,
		},
		{
			name: "inline marks bold and code",
			in:   "use `maxPoolSize` and **stop**",
			want: `{"content":[{"content":[{"text":"use ","type":"text"},{"marks":[{"type":"code"}],"text":"maxPoolSize","type":"text"},{"text":" and ","type":"text"},{"marks":[{"type":"strong"}],"text":"stop","type":"text"}],"type":"paragraph"}],"type":"doc","version":1}`,
		},
		{
			name: "link becomes text with link mark",
			in:   "[docs](https://x.dev)",
			want: `{"content":[{"content":[{"marks":[{"attrs":{"href":"https://x.dev"},"type":"link"}],"text":"docs","type":"text"}],"type":"paragraph"}],"type":"doc","version":1}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.JSONEq(t, tc.want, toJSON(t, markdownToADF(tc.in)))
		})
	}
}

// A sibling node must not inherit the previous sibling's emphasis.
func TestMarkdownToADFMarksDoNotLeakAcrossSiblings(t *testing.T) {
	got := toJSON(t, markdownToADF("*a* b"))
	want := `{"content":[{"content":[{"marks":[{"type":"em"}],"text":"a","type":"text"},{"text":" b","type":"text"}],"type":"paragraph"}],"type":"doc","version":1}`
	assert.JSONEq(t, want, got)
}
