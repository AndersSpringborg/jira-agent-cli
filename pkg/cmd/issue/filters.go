package issue

import (
	"fmt"
	"strings"
)

func statusJQL(statuses string) string {
	var quoted []string
	for _, status := range strings.Split(statuses, ",") {
		status = strings.TrimSpace(status)
		if status != "" {
			quoted = append(quoted, fmt.Sprintf("%q", status))
		}
	}
	switch len(quoted) {
	case 0:
		return ""
	case 1:
		return "status = " + quoted[0]
	default:
		return "status in (" + strings.Join(quoted, ", ") + ")"
	}
}
