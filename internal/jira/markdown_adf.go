package jira

import (
	"strings"

	bf "github.com/russross/blackfriday/v2"
)

// markdownToADF parses CommonMark text and renders it as an Atlassian Document
// Format (ADF) doc, the body shape Jira Cloud REST API v3 requires.
//
// CommonMark is a superset of plain prose, so ordinary text round-trips
// unchanged. Headings, bullet/ordered lists, fenced code blocks, blockquotes,
// rules and inline marks (bold, italic, strike, inline code, links) are
// mapped to their ADF equivalents. Unsupported constructs degrade to text.
func markdownToADF(text string) map[string]any {
	doc := map[string]any{"type": "doc", "version": 1}
	if text == "" {
		doc["content"] = []any{}
		return doc
	}

	root := bf.New(bf.WithExtensions(bf.CommonExtensions)).Parse([]byte(text))
	content := blockNodes(root)
	if content == nil {
		content = []any{}
	}
	doc["content"] = content
	return doc
}

// blockNodes maps the block-level children of parent into ADF block nodes.
func blockNodes(parent *bf.Node) []any {
	var out []any
	for n := parent.FirstChild; n != nil; n = n.Next {
		if node := blockNode(n); node != nil {
			out = append(out, node)
		}
	}
	return out
}

func blockNode(n *bf.Node) map[string]any {
	switch n.Type {
	case bf.Heading:
		return map[string]any{
			"type":    "heading",
			"attrs":   map[string]any{"level": n.Level},
			"content": inlineNodes(n, nil),
		}
	case bf.Paragraph:
		return map[string]any{"type": "paragraph", "content": inlineNodes(n, nil)}
	case bf.BlockQuote:
		return map[string]any{"type": "blockquote", "content": blockNodes(n)}
	case bf.List:
		listType := "bulletList"
		if n.ListFlags&bf.ListTypeOrdered != 0 {
			listType = "orderedList"
		}
		return map[string]any{"type": listType, "content": listItems(n)}
	case bf.CodeBlock:
		block := map[string]any{
			"type":    "codeBlock",
			"content": []any{textNode(strings.TrimRight(string(n.Literal), "\n"), nil)},
		}
		if lang := codeLanguage(n.Info); lang != "" {
			block["attrs"] = map[string]any{"language": lang}
		}
		return block
	case bf.HorizontalRule:
		return map[string]any{"type": "rule"}
	default:
		return nil
	}
}

func listItems(list *bf.Node) []any {
	var items []any
	for item := list.FirstChild; item != nil; item = item.Next {
		if item.Type != bf.Item {
			continue
		}
		items = append(items, map[string]any{"type": "listItem", "content": blockNodes(item)})
	}
	return items
}

// inlineNodes maps inline children of parent into ADF inline nodes, carrying
// the active marks (bold, italic, ...) down through nested emphasis.
func inlineNodes(parent *bf.Node, marks []map[string]any) []any {
	var out []any
	for n := parent.FirstChild; n != nil; n = n.Next {
		out = append(out, inlineNode(n, marks)...)
	}
	return out
}

func inlineNode(n *bf.Node, marks []map[string]any) []any {
	switch n.Type {
	case bf.Text:
		if len(n.Literal) == 0 {
			return nil // blackfriday emits empty text nodes around inline marks
		}
		return []any{textNode(string(n.Literal), marks)}
	case bf.Code:
		return []any{textNode(string(n.Literal), addMark(marks, map[string]any{"type": "code"}))}
	case bf.Emph:
		return inlineNodes(n, addMark(marks, map[string]any{"type": "em"}))
	case bf.Strong:
		return inlineNodes(n, addMark(marks, map[string]any{"type": "strong"}))
	case bf.Del:
		return inlineNodes(n, addMark(marks, map[string]any{"type": "strike"}))
	case bf.Link:
		link := map[string]any{"type": "link", "attrs": map[string]any{"href": string(n.Destination)}}
		return inlineNodes(n, addMark(marks, link))
	case bf.Hardbreak:
		return []any{map[string]any{"type": "hardBreak"}}
	case bf.Softbreak:
		return []any{textNode(" ", marks)}
	default:
		return nil
	}
}

func textNode(text string, marks []map[string]any) map[string]any {
	node := map[string]any{"type": "text", "text": text}
	if len(marks) > 0 {
		m := make([]any, len(marks))
		for i, mark := range marks {
			m[i] = mark
		}
		node["marks"] = m
	}
	return node
}

// addMark returns a new slice with mark appended, never mutating the parent's
// marks (sibling inline nodes must not inherit each other's emphasis).
func addMark(marks []map[string]any, mark map[string]any) []map[string]any {
	next := make([]map[string]any, len(marks), len(marks)+1)
	copy(next, marks)
	return append(next, mark)
}

// codeLanguage extracts the language token from a fenced code block info string
// ("go", "json mytitle" -> "go"); empty for indented blocks.
func codeLanguage(info []byte) string {
	s := strings.TrimSpace(string(info))
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
