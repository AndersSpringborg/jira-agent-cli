package jira

import "strings"

// textToADF wraps a plain-text/markdown-ish string into the minimal Atlassian
// Document Format required by the Jira Cloud REST API v3 description field.
//
// Blank lines split paragraphs. Single newlines inside a paragraph become
// hardBreak nodes so the line break is preserved on render.
func textToADF(text string) map[string]any {
	doc := map[string]any{
		"type":    "doc",
		"version": 1,
	}
	if text == "" {
		doc["content"] = []any{}
		return doc
	}

	paragraphs := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n")
	content := make([]any, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.Trim(p, "\n")
		if p == "" {
			continue
		}
		lines := strings.Split(p, "\n")
		inline := make([]any, 0, len(lines)*2)
		for i, line := range lines {
			if i > 0 {
				inline = append(inline, map[string]any{"type": "hardBreak"})
			}
			if line != "" {
				inline = append(inline, map[string]any{"type": "text", "text": line})
			}
		}
		content = append(content, map[string]any{
			"type":    "paragraph",
			"content": inline,
		})
	}
	if len(content) == 0 {
		content = []any{map[string]any{"type": "paragraph", "content": []any{}}}
	}
	doc["content"] = content
	return doc
}
