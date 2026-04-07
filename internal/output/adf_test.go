package output_test

import (
	"bytes"
	"testing"

	"AndersSpringborg/jira-cli/internal/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ADFToMarkdown unit tests ---

func TestADFToMarkdown_Nil(t *testing.T) {
	assert.Equal(t, "", output.ADFToMarkdown(nil))
}

func TestADFToMarkdown_PlainString(t *testing.T) {
	assert.Equal(t, "hello world", output.ADFToMarkdown("hello world"))
}

func TestADFToMarkdown_EmptyString(t *testing.T) {
	assert.Equal(t, "", output.ADFToMarkdown(""))
}

func TestADFToMarkdown_SimpleADF(t *testing.T) {
	// A minimal ADF document: one paragraph with plain text
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Hello from ADF",
					},
				},
			},
		},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Contains(t, result, "Hello from ADF")
	assert.NotContains(t, result, "map[")
}

func TestADFToMarkdown_MultiParagraph(t *testing.T) {
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "First paragraph",
					},
				},
			},
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Second paragraph",
					},
				},
			},
		},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Contains(t, result, "First paragraph")
	assert.Contains(t, result, "Second paragraph")
	assert.NotContains(t, result, "map[")
}

func TestADFToMarkdown_WithBoldText(t *testing.T) {
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Important",
						"marks": []any{
							map[string]any{"type": "strong"},
						},
					},
				},
			},
		},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Contains(t, result, "**Important**")
}

func TestADFToMarkdown_WithHeading(t *testing.T) {
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{
			map[string]any{
				"type":  "heading",
				"attrs": map[string]any{"level": float64(2)},
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Section Title",
					},
				},
			},
		},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Contains(t, result, "## Section Title")
}

func TestADFToMarkdown_WithBulletList(t *testing.T) {
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{
			map[string]any{
				"type": "bulletList",
				"content": []any{
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": "Item one"},
								},
							},
						},
					},
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": "Item two"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Contains(t, result, "- Item one")
	assert.Contains(t, result, "- Item two")
}

func TestADFToMarkdown_NonADFMap(t *testing.T) {
	// A map that doesn't have type "doc" should fall through to fmt
	m := map[string]any{"foo": "bar"}
	result := output.ADFToMarkdown(m)
	assert.NotEqual(t, "", result)
	// Should not panic, just produce some string representation
}

func TestADFToMarkdown_EmptyADFDoc(t *testing.T) {
	adfDoc := map[string]any{
		"version": float64(1),
		"type":    "doc",
		"content": []any{},
	}

	result := output.ADFToMarkdown(adfDoc)
	assert.Equal(t, "", result)
}

// --- Integration: MarkdownDriver.Item with ADF description ---

func TestMarkdownDriver_Item_ADFDescription(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"key": "PROJ-100",
		"fields": map[string]any{
			"summary": "ADF description test",
			"status":  map[string]any{"name": "Open"},
			"description": map[string]any{
				"version": float64(1),
				"type":    "doc",
				"content": []any{
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{
								"type": "text",
								"text": "When a user tries to log in with SSO, they are redirected to a blank page.",
							},
						},
					},
					map[string]any{
						"type":  "heading",
						"attrs": map[string]any{"level": float64(2)},
						"content": []any{
							map[string]any{
								"type": "text",
								"text": "Steps to reproduce",
							},
						},
					},
					map[string]any{
						"type": "orderedList",
						"content": []any{
							map[string]any{
								"type": "listItem",
								"content": []any{
									map[string]any{
										"type": "paragraph",
										"content": []any{
											map[string]any{"type": "text", "text": "Go to login page"},
										},
									},
								},
							},
							map[string]any{
								"type": "listItem",
								"content": []any{
									map[string]any{
										"type": "paragraph",
										"content": []any{
											map[string]any{"type": "text", "text": "Click SSO button"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	out := buf.String()

	// Should have proper heading
	assert.Contains(t, out, "## PROJ-100: ADF description test")

	// Description should be rendered as markdown, NOT as map[content:[...]]
	assert.NotContains(t, out, "map[")
	assert.Contains(t, out, "### Description")
	assert.Contains(t, out, "redirected to a blank page")
	assert.Contains(t, out, "## Steps to reproduce")
	assert.Contains(t, out, "1. Go to login page")
	assert.Contains(t, out, "2. Click SSO button")
}

// --- Integration: renderComments with ADF comment bodies ---

func TestMarkdownDriver_Item_ADFCommentBody(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"key": "PROJ-200",
		"fields": map[string]any{
			"summary": "Comment body test",
			"comment": map[string]any{
				"comments": []any{
					map[string]any{
						"author":  map[string]any{"displayName": "Alice"},
						"created": "2024-03-01",
						"body": map[string]any{
							"version": float64(1),
							"type":    "doc",
							"content": []any{
								map[string]any{
									"type": "paragraph",
									"content": []any{
										map[string]any{
											"type": "text",
											"text": "I can confirm this issue.",
										},
									},
								},
								map[string]any{
									"type": "paragraph",
									"content": []any{
										map[string]any{
											"type": "text",
											"text": "Bold part",
											"marks": []any{
												map[string]any{"type": "strong"},
											},
										},
									},
								},
							},
						},
					},
					map[string]any{
						"author":  map[string]any{"displayName": "Bob"},
						"created": "2024-03-02",
						"body":    "Plain text comment from v2 API",
					},
				},
			},
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	out := buf.String()

	// ADF comment should be rendered as markdown
	assert.NotContains(t, out, "map[")
	assert.Contains(t, out, "**Alice**")
	assert.Contains(t, out, "I can confirm this issue.")
	assert.Contains(t, out, "**Bold part**")

	// Plain text comment should pass through unchanged
	assert.Contains(t, out, "**Bob**")
	assert.Contains(t, out, "Plain text comment from v2 API")
}

// --- Regression: plain string descriptions still work ---

func TestMarkdownDriver_Item_PlainStringDescription(t *testing.T) {
	var buf bytes.Buffer
	d := output.NewDriverWithWriter(output.FormatMarkdown, &buf)

	data := map[string]any{
		"key": "PROJ-300",
		"fields": map[string]any{
			"summary":     "Plain description",
			"description": "This is a plain text description from the v2 API.",
		},
	}

	err := d.Item("Issue", data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "### Description")
	assert.Contains(t, out, "This is a plain text description from the v2 API.")
}
