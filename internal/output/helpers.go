package output

import (
	"fmt"
	"strings"
)

// FormatValue extracts a display string from a value that might be
// a nested map (e.g. {"name": "High"} or {"displayName": "Jane"}).
// Returns the string representation for display.
func FormatValue(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case map[string]any:
		// Common Jira patterns: {"name": "..."} or {"displayName": "..."}
		if name, ok := val["name"].(string); ok && name != "" {
			return name
		}
		if name, ok := val["displayName"].(string); ok && name != "" {
			return name
		}
		if key, ok := val["key"].(string); ok && key != "" {
			return key
		}
		if value, ok := val["value"].(string); ok && value != "" {
			return value
		}
		return ""
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			s := FormatValue(item)
			if s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	case []string:
		return strings.Join(val, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// NormalizeFields parses a comma-separated column string.
// If the user provided columns, it splits and trims them.
// Otherwise it returns the defaults.
func NormalizeFields(userColumns string, defaults []string) []string {
	if userColumns == "" {
		return defaults
	}
	parts := strings.Split(userColumns, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return defaults
	}
	return result
}

// Green returns the string as-is. No ANSI colors in AI-first output.
func Green(s string) string { return s }

// Red returns the string as-is. No ANSI colors in AI-first output.
func Red(s string) string { return s }

// Dim returns the string as-is. No ANSI colors in AI-first output.
func Dim(s string) string { return s }
