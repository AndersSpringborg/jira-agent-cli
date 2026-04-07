package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONDriver renders output as JSON.
// This is the default driver -- designed for machine consumption,
// jq pipelines, and LLM tool-use parsing.
type JSONDriver struct {
	w io.Writer
}

func (d *JSONDriver) Item(_ string, data map[string]any) error {
	return d.encode(data)
}

func (d *JSONDriver) List(_ string, _ []string, rows []map[string]any) error {
	return d.encode(rows)
}

func (d *JSONDriver) Raw(data any) error {
	return d.encode(data)
}

func (d *JSONDriver) Message(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return d.encode(map[string]any{"message": msg})
}

func (d *JSONDriver) Error(err error) error {
	return d.encode(map[string]any{"error": err.Error()})
}

func (d *JSONDriver) encode(v any) error {
	enc := json.NewEncoder(d.w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
