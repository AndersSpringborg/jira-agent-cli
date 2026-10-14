//go:build !windows

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"
	"AndersSpringborg/jira-cli/internal/jira"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safeBoundaryJQL = `labels = "safe"`

// boundaryHarness runs the real root command against a fake Jira Server,
// substituting only the privilege check and the boundary directory.
type boundaryHarness struct {
	t          *testing.T
	dir        string
	privileged bool

	mu  sync.Mutex
	jql []string
}

func newBoundaryHarness(t *testing.T) *boundaryHarness {
	t.Helper()
	h := &boundaryHarness{t: t, dir: filepath.Join(t.TempDir(), "boundaries")}
	server := httptest.NewServer(http.HandlerFunc(h.serveJira))
	t.Cleanup(server.Close)

	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", server.URL)
	t.Setenv("JIRA_TOKEN", "secret-token")
	t.Setenv("JIRA_AUTH_TYPE", "pat")
	return h
}

func (h *boundaryHarness) serveJira(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	issue := func(key string, labels ...any) map[string]any {
		return map[string]any{"key": key, "id": "1" + strings.Split(key, "-")[1], "fields": map[string]any{
			"summary": "Summary of " + key, "project": map[string]any{"key": "CER"}, "labels": labels,
		}}
	}
	switch r.URL.Path {
	case "/rest/api/2/search":
		h.jql = append(h.jql, r.URL.Query().Get("jql"))
		issues := []any{issue("CER-1", "safe")}
		if strings.Contains(r.URL.Query().Get("jql"), "CER-2") {
			issues = []any{issue("CER-2", "internal")}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": issues})
	case "/rest/api/2/issue/CER-2":
		_ = json.NewEncoder(w).Encode(issue("CER-2", "internal"))
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}
}

func (h *boundaryHarness) receivedJQL() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.jql...)
}

func (h *boundaryHarness) run(args ...string) (string, error) {
	h.t.Helper()
	privileged := h.privileged
	root := newRootCmd("test", "today", &cmdutil.Factory{
		Boundaries: config.NewBoundaryStore(h.dir, "alice", config.OwnerGuard{
			TrustedUID: uint32(os.Getuid()),
			Privileged: func() bool { return privileged },
		}),
	})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func (h *boundaryHarness) asSudo(args ...string) {
	h.t.Helper()
	h.privileged = true
	defer func() { h.privileged = false }()
	_, err := h.run(args...)
	require.NoError(h.t, err)
}

func TestBoundarySetWithoutSudoTellsAgentToAskTheUser(t *testing.T) {
	h := newBoundaryHarness(t)

	_, err := h.run("boundary", "set", "--label", "safe", "--project", "CER")

	require.ErrorIs(t, err, config.ErrBoundaryRequiresSudo)
	assert.Contains(t, err.Error(), "Ask the user to run this command")
	assert.Contains(t, err.Error(), "Never run it yourself")
	assert.Contains(t, err.Error(), "never try to bypass")
	assert.Contains(t, err.Error(), "sudo jira boundary set --label safe --project CER")
	_, statErr := os.Stat(filepath.Join(h.dir, "alice.yml"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestBoundaryClearWithoutSudoIsRejected(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe")

	_, err := h.run("boundary", "clear")

	require.ErrorIs(t, err, config.ErrBoundaryRequiresSudo)
	assert.Contains(t, err.Error(), "sudo jira boundary clear")
}

func TestBoundarySetWithSudoIsReadableWithoutSudo(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe", "--project", "CER")

	out, err := h.run("boundary", "show")

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"active": true,
		"projects": ["CER"],
		"labels": ["safe"],
		"jql": "project in (\"CER\") AND labels = \"safe\"",
		"path": `+jsonString(filepath.Join(h.dir, "alice.yml"))+`,
		"protected": true
	}`, out)
}

func TestContextChangesWithoutSudoButQueriesStayInsideBoundary(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe")
	require.NoError(t, config.Save(&config.Config{
		DefaultProfile: "default",
		Profiles:       map[string]*config.Profile{"default": {Name: "default"}},
	}))

	_, err := h.run("context", "set", "work", "--project", "CER", "--epic", "CER-33")
	require.NoError(t, err, "the context is the agent's working focus and needs no sudo")
	_, err = h.run("context", "use", "work")
	require.NoError(t, err)

	_, err = h.run("issue", "list")
	require.NoError(t, err)
	_, err = h.run("issue", "list", "--epic", "CER-99")
	require.NoError(t, err, "queries outside the context are allowed")
	_, err = h.run("search", "jql", "project = OPS")
	require.NoError(t, err)

	jql := h.receivedJQL()
	require.Len(t, jql, 3)
	assert.Equal(t, `(`+safeBoundaryJQL+`) AND (project = CER AND ("Epic Link" = "CER-33" OR parent = "CER-33")) ORDER BY updated DESC`, jql[0])
	assert.Equal(t, `(`+safeBoundaryJQL+`) AND (project = CER AND ("Epic Link" = "CER-99" OR parent = "CER-99")) ORDER BY updated DESC`, jql[1])
	assert.Equal(t, `(`+safeBoundaryJQL+`) AND (project = OPS)`, jql[2])
}

func TestIssueOutsideBoundaryCannotBeViewed(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe")

	out, err := h.run("issue", "view", "CER-2")

	require.ErrorIs(t, err, jira.ErrOutsideBoundary)
	assert.NotContains(t, out, "Summary of CER-2")
}

func TestRawAPIIsRefusedInsideBoundaryButAllowedWithSudo(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe")

	_, err := h.run("api", "/rest/api/2/issue/CER-2")
	require.ErrorIs(t, err, jira.ErrOutsideBoundary)
	assert.Contains(t, err.Error(), "sudo")

	h.asSudo("api", "/rest/api/2/issue/CER-2")
}

func TestBoundaryClearedWithSudoLiftsRestriction(t *testing.T) {
	h := newBoundaryHarness(t)
	h.asSudo("boundary", "set", "--label", "safe")
	h.asSudo("boundary", "clear")

	_, err := h.run("search", "jql", "project = OPS")

	require.NoError(t, err)
	assert.Equal(t, []string{"project = OPS"}, h.receivedJQL())
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
