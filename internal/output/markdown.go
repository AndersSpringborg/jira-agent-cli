package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// MarkdownDriver renders output as structured markdown.
// Designed for LLM consumption: headers, key-value tables,
// and clear section structure that language models parse well.
type MarkdownDriver struct {
	w io.Writer
}

func (d *MarkdownDriver) Item(title string, data map[string]any) error {
	// Build a heading from the title and key/summary if available
	heading := title
	if key, ok := data["key"].(string); ok {
		heading = key
		if fields, ok := data["fields"].(map[string]any); ok {
			if summary, ok := fields["summary"].(string); ok {
				heading = fmt.Sprintf("%s: %s", key, summary)
			}
		}
	}
	if _, err := fmt.Fprintf(d.w, "## %s\n\n", heading); err != nil {
		return err
	}

	// Render fields as a key-value table
	if err := d.renderFieldTable(data); err != nil {
		return err
	}

	// Render description if present (nested under fields)
	if fields, ok := data["fields"].(map[string]any); ok {
		if desc := fields["description"]; desc != nil {
			if _, err := fmt.Fprintf(d.w, "\n### Description\n\n%s\n", ADFToMarkdown(desc)); err != nil {
				return err
			}
		}

		// Render comments if present
		if err := d.renderComments(fields); err != nil {
			return err
		}
	}

	return nil
}

func (d *MarkdownDriver) List(title string, columns []string, rows []map[string]any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintf(d.w, "No %s found.\n", strings.ToLower(title))
		return err
	}

	if _, err := fmt.Fprintf(d.w, "## %s (%d)\n\n", title, len(rows)); err != nil {
		return err
	}

	// Markdown table header
	if _, err := fmt.Fprintf(d.w, "| %s |\n", strings.Join(columns, " | ")); err != nil {
		return err
	}
	seps := make([]string, len(columns))
	for i := range seps {
		seps[i] = "---"
	}
	if _, err := fmt.Fprintf(d.w, "| %s |\n", strings.Join(seps, " | ")); err != nil {
		return err
	}

	// Markdown table rows
	for _, row := range rows {
		vals := make([]string, len(columns))
		for i, col := range columns {
			vals[i] = FormatValue(row[col])
		}
		if _, err := fmt.Fprintf(d.w, "| %s |\n", strings.Join(vals, " | ")); err != nil {
			return err
		}
	}

	return nil
}

func (d *MarkdownDriver) Raw(data any) error {
	// For markdown raw, fall back to JSON driver since the user
	// explicitly asked for the raw API response.
	json := &JSONDriver{w: d.w}
	return json.Raw(data)
}

func (d *MarkdownDriver) Message(format string, args ...any) error {
	_, err := fmt.Fprintf(d.w, format+"\n", args...)
	return err
}

func (d *MarkdownDriver) Error(err error) error {
	_, werr := fmt.Fprintf(os.Stderr, "**Error:** %s\n", err.Error())
	return werr
}

// renderFieldTable renders the top-level scalar fields of an item
// as a markdown key-value table.
func (d *MarkdownDriver) renderFieldTable(data map[string]any) error {
	// Collect fields to render. If there's a nested "fields" object
	// (Jira issue structure), extract displayable fields from it.
	fields := extractDisplayFields(data)
	if len(fields) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(d.w, "| Field | Value |\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(d.w, "| --- | --- |\n"); err != nil {
		return err
	}
	for _, kv := range fields {
		if _, err := fmt.Fprintf(d.w, "| %s | %s |\n", kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

// renderComments renders issue comments as markdown sections.
func (d *MarkdownDriver) renderComments(fields map[string]any) error {
	commentField, ok := fields["comment"].(map[string]any)
	if !ok {
		return nil
	}
	commentList, ok := commentField["comments"].([]any)
	if !ok || len(commentList) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(d.w, "\n### Comments (%d)\n", len(commentList)); err != nil {
		return err
	}
	for _, c := range commentList {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		author := "Unknown"
		if a, ok := cm["author"].(map[string]any); ok {
			author = FormatValue(a)
		}
		created := ""
		if c, ok := cm["created"].(string); ok {
			created = fmt.Sprintf(" (%s)", c)
		}
		if _, err := fmt.Fprintf(d.w, "\n**%s**%s:\n%s\n", author, created, ADFToMarkdown(cm["body"])); err != nil {
			return err
		}
	}
	return nil
}

// extractDisplayFields pulls scalar/displayable fields from a data map.
// Returns ordered key-value pairs for rendering.
func extractDisplayFields(data map[string]any) [][2]string {
	var result [][2]string

	// If the data has a Jira issue structure (key + fields), use that.
	if _, hasKey := data["key"]; hasKey {
		if key, ok := data["key"].(string); ok {
			result = append(result, [2]string{"Key", key})
		}
	}

	// Extract from nested "fields" if present (Jira issue structure)
	nested, hasFields := data["fields"].(map[string]any)
	source := data
	if hasFields {
		source = nested
	}

	// Ordered list of well-known fields to display
	displayOrder := []struct {
		key   string
		label string
	}{
		{"summary", "Summary"},
		{"issuetype", "Type"},
		{"status", "Status"},
		{"priority", "Priority"},
		{"assignee", "Assignee"},
		{"reporter", "Reporter"},
		{"labels", "Labels"},
		{"created", "Created"},
		{"updated", "Updated"},
		{"resolution", "Resolution"},
		// Generic fields for non-issue objects
		{"name", "Name"},
		{"type", "Type"},
		{"id", "ID"},
		{"state", "State"},
		{"startDate", "Start Date"},
		{"endDate", "End Date"},
		{"displayName", "Display Name"},
		{"emailAddress", "Email"},
		{"accountId", "Account ID"},
		{"projectTypeKey", "Project Type"},
	}

	seen := map[string]bool{}
	if _, hasKey := data["key"]; hasKey {
		seen["key"] = true
	}

	for _, f := range displayOrder {
		v, ok := source[f.key]
		if !ok || v == nil {
			continue
		}
		// Skip description and comment -- rendered separately
		if f.key == "description" || f.key == "comment" {
			continue
		}
		formatted := FormatValue(v)
		if formatted == "" {
			continue
		}
		result = append(result, [2]string{f.label, formatted})
		seen[f.key] = true
	}

	// Include any remaining fields not in the display order
	// (skip complex nested objects like description, comment)
	for k, v := range source {
		if seen[k] || k == "description" || k == "comment" {
			continue
		}
		formatted := FormatValue(v)
		if formatted != "" {
			result = append(result, [2]string{k, formatted})
		}
	}

	return result
}
