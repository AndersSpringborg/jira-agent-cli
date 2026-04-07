// Package output provides display driver abstractions for CLI output.
//
// Commands push data through a DisplayDriver interface rather than
// formatting output directly. This makes the CLI AI-first: the default
// driver produces JSON (machine-parseable, jq-friendly), while the
// markdown driver produces structured markdown optimized for LLM consumption.
package output

import (
	"fmt"
	"io"
	"os"
)

// DisplayDriver defines the interface for rendering CLI output.
// Each method corresponds to a common output pattern used by commands.
type DisplayDriver interface {
	// Item renders a single object (e.g. issue view, board get, me).
	// title is a human-readable label like "Issue" or "Board".
	// data is the object to display.
	Item(title string, data map[string]any) error

	// List renders a collection of objects as rows (e.g. board list, search results).
	// title is a human-readable label like "Boards" or "Issues".
	// columns defines which fields to display and their order.
	// rows is the data to display.
	List(title string, columns []string, rows []map[string]any) error

	// Raw renders arbitrary data with no transformation.
	// Used for --raw flag: dumps the exact API response.
	Raw(data any) error

	// Message renders a simple text message (e.g. "Created issue: PROJ-123").
	Message(format string, args ...any) error

	// Error renders an error message.
	Error(err error) error
}

// Format represents an output format type.
type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// NewDriver creates a DisplayDriver for the given format.
// Writes output to os.Stdout by default.
func NewDriver(format Format) DisplayDriver {
	return NewDriverWithWriter(format, os.Stdout)
}

// NewDriverWithWriter creates a DisplayDriver for the given format,
// writing to the specified writer.
func NewDriverWithWriter(format Format, w io.Writer) DisplayDriver {
	switch format {
	case FormatMarkdown:
		return &MarkdownDriver{w: w}
	case FormatJSON:
		return &JSONDriver{w: w}
	default:
		return &JSONDriver{w: w}
	}
}

// defaultWriter returns the default output writer (stdout).
func defaultWriter() io.Writer {
	return os.Stdout
}

// ParseFormat parses a format string into a Format constant.
// Returns an error for unrecognized formats.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "json", "":
		return FormatJSON, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unknown output format %q, supported: json, markdown", s)
	}
}
