package output

import (
	"encoding/json"
	"fmt"

	"AndersSpringborg/jira-cli/pkg/adf"
)

// ADFToMarkdown converts an ADF value to a markdown string.
//
// The Jira v3 API returns descriptions and comment bodies as Atlassian
// Document Format (ADF) — a nested JSON structure with type "doc".
// When the value is a map with type "doc", this function deserializes
// it into the typed ADF struct and runs the markdown translator.
//
// For plain strings (v2 API or already-converted values), it returns
// the string as-is. For nil, it returns an empty string.
func ADFToMarkdown(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val

	case map[string]any:
		// Check if this looks like an ADF document
		if docType, ok := val["type"].(string); ok && docType == "doc" {
			return convertADF(val)
		}
		// Not ADF — fall back to compact JSON
		return marshalFallback(v)

	default:
		return marshalFallback(v)
	}
}

// convertADF marshals a map[string]any to JSON, unmarshals into the
// typed *adf.ADF struct, and runs the markdown translator.
func convertADF(m map[string]any) string {
	data, err := json.Marshal(m)
	if err != nil {
		return marshalFallback(m)
	}

	var doc adf.ADF
	if err := json.Unmarshal(data, &doc); err != nil {
		return marshalFallback(m)
	}

	tr := adf.NewTranslator(&doc, adf.NewMarkdownTranslator())
	return tr.Translate()
}

// marshalFallback produces compact JSON for non-string, non-ADF values.
// This avoids Go's default fmt.Sprintf("%v") which produces unreadable
// map[key:value] output for nested structures.
func marshalFallback(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}
