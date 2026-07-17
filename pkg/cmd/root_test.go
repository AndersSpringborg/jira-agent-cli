package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutePrintsFailingCommandHelpOnError(t *testing.T) {
	root := NewRootCmd("test", "today")
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{"issue", "view"})

	err := Execute(root)
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "accepts 1 arg(s), received 0")
	assert.Contains(t, stderr.String(), "Usage:")
	assert.Contains(t, stderr.String(), "jira issue view <issue-key>")
	assert.Contains(t, stderr.String(), "--comments")
}

func TestDebugFlagWritesJiraResponseToStderr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/myself", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"accountId": "user-123"})
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", server.URL)
	t.Setenv("JIRA_TOKEN", "secret-token")
	t.Setenv("JIRA_AUTH_TYPE", "pat")

	root := NewRootCmd("test", "today")
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{"ping", "--debug"})

	require.NoError(t, root.Execute())
	assert.Contains(t, stderr.String(), "--- jira debug response ---")
	assert.Contains(t, stderr.String(), "GET "+server.URL+"/rest/api/2/myself")
	assert.Contains(t, stderr.String(), `{"accountId":"user-123"}`)
	assert.NotContains(t, stderr.String(), "secret-token")
}

func TestNamedContextsRetainProjectAndProfileWhenSwitched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Save(&config.Config{
		DefaultProfile: "default",
		Profiles: map[string]*config.Profile{
			"default": {Name: "default"},
			"trifork": {Name: "trifork"},
		},
	}))

	executeArgs := func(args ...string) {
		t.Helper()
		root := NewRootCmd("test", "today")
		root.SetArgs(args)
		require.NoError(t, root.Execute())
	}

	executeArgs("context", "set", "cai", "--project", "CAI", "--profile", "trifork")
	executeArgs("context", "set", "personal", "--project", "HOME", "--profile", "default")
	executeArgs("context", "use", "cai")

	configBytes, err := os.ReadFile(os.Getenv("HOME") + "/.config/jira-cli/config.yml")
	require.NoError(t, err)
	configYAML := string(configBytes)
	assert.Contains(t, configYAML, "active_context: cai")
	assert.Contains(t, configYAML, "contexts:\n    - name: cai")
	assert.NotContains(t, configYAML, "contexts:\n    cai:")
	assert.Contains(t, configYAML, "profile: trifork")
	assert.Contains(t, configYAML, "project: CAI")
	assert.Contains(t, configYAML, "- name: personal")
	assert.Contains(t, configYAML, "project: HOME")

	profile, err := (&cmdutil.Factory{}).LoadProfile()
	require.NoError(t, err)
	assert.Equal(t, "trifork", profile.Name)
	require.NotNil(t, profile.Context)
	assert.Equal(t, "CAI", profile.Context.Project)
}
