package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCommandIsRegistered(t *testing.T) {
	root := NewRootCmd("1.0.0", "today")
	command, _, err := root.Find([]string{"update"})
	require.NoError(t, err)
	assert.Equal(t, "update", command.Name())
	assert.Contains(t, command.Long, "npm installations are upgraded through npm")
}

func TestUpdateUsesNPMForNPMInstallations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a shell script as a fake npm executable")
	}
	binDir := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "npm-args")
	npm := filepath.Join(binDir, "npm")
	require.NoError(t, os.WriteFile(npm, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$NPM_ARGS_FILE\"\n"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("JIRA_CLI_INSTALL_METHOD", "npm")
	t.Setenv("NPM_ARGS_FILE", argsFile)

	root := NewRootCmd("1.0.0", "today")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"update"})
	require.NoError(t, root.Execute())

	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	assert.Equal(t, "install -g @888aaen/jira-cli@latest\n", string(args))
	assert.JSONEq(t, `{"method":"npm","previousVersion":"1.0.0","updated":true}`, stdout.String())
}

func TestIssueGetIncludesAttachmentsByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "*navigable,attachment", r.URL.Query().Get("fields"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "TEST-1",
			"fields": map[string]any{
				"summary": "Issue with a file",
				"attachment": []any{map[string]any{
					"id": "10042", "filename": "screenshot.png",
				}},
			},
		})
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", server.URL)
	t.Setenv("JIRA_TOKEN", "secret-token")
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_AUTH_TYPE", "basic")

	root := NewRootCmd("test", "today")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"issue", "get", "test-1"})
	require.NoError(t, root.Execute())
	assert.Contains(t, stdout.String(), `"filename": "screenshot.png"`)
	assert.Contains(t, stdout.String(), `"action": "downloadAttachment"`)
	assert.Contains(t, stdout.String(), "jira issue attachment download TEST-1 ATTACHMENT_ID --output PATH")
}

func TestIssueAttachmentDownload(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/TEST-1":
			assert.Equal(t, "attachment", r.URL.Query().Get("fields"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "TEST-1",
				"fields": map[string]any{"attachment": []any{map[string]any{
					"id": "10042", "filename": "screenshot.png", "mimeType": "image/png",
					"size": float64(12), "content": server.URL + "/attachment/10042",
				}}},
			})
		case "/attachment/10042":
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "test@example.com", username)
			assert.Equal(t, "secret-token", password)
			_, _ = w.Write([]byte("image bytes"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", server.URL)
	t.Setenv("JIRA_TOKEN", "secret-token")
	t.Setenv("JIRA_EMAIL", "test@example.com")
	t.Setenv("JIRA_AUTH_TYPE", "basic")
	outputPath := filepath.Join(t.TempDir(), "screenshot.png")

	root := NewRootCmd("test", "today")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"issue", "attachment", "download", "test-1", "screenshot.png", "--output", outputPath})
	require.NoError(t, root.Execute())

	contents, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "image bytes", string(contents))
	assert.JSONEq(t, fmt.Sprintf(`{"issueKey":"TEST-1","id":"10042","filename":"screenshot.png","mimeType":"image/png","size":12,"path":%q}`, outputPath), stdout.String())
}

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
