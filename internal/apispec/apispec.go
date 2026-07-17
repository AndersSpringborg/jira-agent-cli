// Package apispec exposes a compact catalog of Jira REST API endpoints
// for the `jira api --list` command. The catalogs are generated from the
// official Atlassian OpenAPI specs by openapi/build-api-index.py and
// embedded into the binary — listing works offline and without auth.
package apispec

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Flavor identifies which Jira REST API dialect a catalog describes.
type Flavor string

const (
	// FlavorCloud is Jira Cloud: platform REST API v3 plus the agile API.
	FlavorCloud Flavor = "cloud"
	// FlavorServer is Jira Server/Data Center: platform REST API v2 plus the agile API.
	FlavorServer Flavor = "server"
)

// Endpoint is one operation in the Jira REST API.
type Endpoint struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

//go:embed index_cloud.json
var cloudIndex []byte

//go:embed index_server.json
var serverIndex []byte

// List returns the endpoints for the given flavor. A non-empty filter keeps
// only endpoints whose method, path, or summary contain every
// whitespace-separated term (case-insensitive).
func List(flavor Flavor, filter string) ([]Endpoint, error) {
	var raw []byte
	switch flavor {
	case FlavorCloud:
		raw = cloudIndex
	case FlavorServer:
		raw = serverIndex
	default:
		return nil, fmt.Errorf("apispec: unknown flavor %q", flavor)
	}

	var endpoints []Endpoint
	if err := json.Unmarshal(raw, &endpoints); err != nil {
		return nil, fmt.Errorf("apispec: decode embedded %s index: %w", flavor, err)
	}

	terms := strings.Fields(strings.ToLower(filter))
	if len(terms) == 0 {
		return endpoints, nil
	}

	filtered := make([]Endpoint, 0, len(endpoints))
	for _, e := range endpoints {
		haystack := strings.ToLower(e.Method + " " + e.Path + " " + e.Summary)
		match := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}
