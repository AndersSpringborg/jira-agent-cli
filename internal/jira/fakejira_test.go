package jira_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"AndersSpringborg/jira-cli/internal/jira"

	"github.com/stretchr/testify/require"
)

// fakeJira is an in-memory Jira Server (REST v2) holding a few realistic
// issues. It deliberately ignores boundary clauses in JQL, returning every
// matching key, so tests prove the client filters results itself.
type fakeJira struct {
	t      *testing.T
	mu     sync.Mutex
	issues map[string]map[string]any
	order  []string
	links  map[string]map[string]any
	// jql records every JQL query received; writes records "METHOD path".
	jql    []string
	writes []string
	bodies []map[string]any
}

var issueKeyListPattern = regexp.MustCompile(`issuekey in \(([^)]*)\)`)

func issue(key, project string, labels []string, extra map[string]any) map[string]any {
	labelValues := make([]any, len(labels))
	for i, l := range labels {
		labelValues[i] = l
	}
	fields := map[string]any{
		"summary": "Summary of " + key,
		"project": map[string]any{"key": project},
		"labels":  labelValues,
		"status":  map[string]any{"name": "To Do"},
	}
	for k, v := range extra {
		fields[k] = v
	}
	return map[string]any{"id": "1" + strings.Split(key, "-")[1], "key": key, "fields": fields}
}

func linkedIssue(key string) map[string]any {
	return map[string]any{"key": key, "fields": map[string]any{"summary": "Secret summary of " + key}}
}

func newFakeJira(t *testing.T) *fakeJira {
	f := &fakeJira{t: t, issues: map[string]map[string]any{}, links: map[string]map[string]any{}}
	f.add(issue("CER-1", "CER", []string{"safe"}, map[string]any{
		"issuelinks": []any{
			map[string]any{"id": "100", "type": map[string]any{"name": "Blocks"}, "outwardIssue": linkedIssue("CER-3")},
			map[string]any{"id": "101", "type": map[string]any{"name": "Blocks"}, "inwardIssue": linkedIssue("CER-2")},
		},
		"subtasks": []any{linkedIssue("CER-2"), linkedIssue("CER-3")},
		"parent":   linkedIssue("OPS-1"),
	}))
	f.add(issue("CER-2", "CER", []string{"internal"}, nil))
	f.add(issue("CER-3", "CER", []string{"safe", "backend"}, nil))
	f.add(issue("OPS-1", "OPS", []string{"safe"}, nil))
	f.links["100"] = map[string]any{"id": "100", "inwardIssue": map[string]any{"key": "CER-1"}, "outwardIssue": map[string]any{"key": "CER-3"}}
	f.links["101"] = map[string]any{"id": "101", "inwardIssue": map[string]any{"key": "CER-2"}, "outwardIssue": map[string]any{"key": "CER-1"}}
	return f
}

func (f *fakeJira) add(issue map[string]any) {
	key := issue["key"].(string)
	f.issues[key] = issue
	f.order = append(f.order, key)
}

func (f *fakeJira) client(t *testing.T) *jira.Client {
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	client, err := jira.NewClient(server.URL, "", "test-token", "pat", 5)
	require.NoError(t, err)
	return client
}

func (f *fakeJira) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	if r.Method != http.MethodGet {
		f.writes = append(f.writes, r.Method+" "+path)
		var body map[string]any
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		f.bodies = append(f.bodies, body)
		if r.Method == http.MethodPost && path == "/rest/api/2/issue" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10100", "key": "CER-100"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch {
	case path == "/rest/api/2/search" || strings.HasSuffix(path, "/issue") && strings.HasPrefix(path, "/rest/agile/1.0/sprint/"):
		jql := r.URL.Query().Get("jql")
		f.jql = append(f.jql, jql)
		f.writeJSON(w, map[string]any{"issues": f.search(jql)})
	case strings.HasPrefix(path, "/rest/api/2/issue/"):
		key := strings.TrimPrefix(path, "/rest/api/2/issue/")
		issue, ok := f.issues[strings.ToUpper(key)]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f.writeJSON(w, issue)
	case strings.HasPrefix(path, "/rest/api/2/issueLink/"):
		f.writeJSON(w, f.links[strings.TrimPrefix(path, "/rest/api/2/issueLink/")])
	case path == "/rest/api/2/project":
		_ = json.NewEncoder(w).Encode([]any{map[string]any{"key": "CER"}, map[string]any{"key": "OPS"}})
	case strings.HasPrefix(path, "/rest/api/2/project/"):
		f.writeJSON(w, map[string]any{"key": strings.TrimPrefix(path, "/rest/api/2/project/")})
	default:
		f.t.Errorf("fake Jira: unexpected GET %s", path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeJira) search(jql string) []any {
	keys := f.order
	if match := issueKeyListPattern.FindStringSubmatch(jql); match != nil {
		keys = nil
		for _, k := range strings.Split(match[1], ",") {
			keys = append(keys, strings.ToUpper(strings.TrimSpace(k)))
		}
	}
	result := []any{}
	for _, k := range keys {
		if issue, ok := f.issues[k]; ok {
			result = append(result, issue)
		}
	}
	return result
}

func (f *fakeJira) writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		f.t.Errorf("fake Jira: encode: %v", err)
	}
}

func issueKeys(data map[string]any) []string {
	issues, _ := data["issues"].([]any)
	keys := make([]string, 0, len(issues))
	for _, raw := range issues {
		keys = append(keys, raw.(map[string]any)["key"].(string))
	}
	return keys
}
