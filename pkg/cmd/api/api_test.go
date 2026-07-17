package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/pkg/cmd/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEnv(t *testing.T, serverURL, authType string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", serverURL)
	t.Setenv("JIRA_TOKEN", "secret-token")
	t.Setenv("JIRA_AUTH_TYPE", authType)
	t.Setenv("JIRABOT_PROFILE", "")
}

func runCmd(t *testing.T, args []string, stdin string) (string, error) {
	t.Helper()
	cmd := api.NewCmd(&cmdutil.Factory{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestGetFullPathPassthrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/rest/agile/1.0/board/42/backlog", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("maxResults"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}))
	defer server.Close()
	setupEnv(t, server.URL, "pat")

	out, err := runCmd(t, []string{"/rest/agile/1.0/board/42/backlog?maxResults=10"}, "")
	require.NoError(t, err)
	assert.Equal(t, "{\"issues\":[]}\n", out)
}

func TestShorthandPathUsesProfileFlavor(t *testing.T) {
	tests := []struct {
		name     string
		authType string
		wantPath string
	}{
		{"cloud gets v3", "basic", "/rest/api/3/issue/PROJ-1"},
		{"on-prem gets v2", "pat", "/rest/api/2/issue/PROJ-1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.wantPath, r.URL.Path)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()
			setupEnv(t, server.URL, tc.authType)

			_, err := runCmd(t, []string{"issue/PROJ-1"}, "")
			require.NoError(t, err)
		})
	}
}

func TestPostWithInlineBodyAndHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "opt-in", r.Header.Get("X-ExperimentalApi"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "hello", body["summary"])
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer server.Close()
	setupEnv(t, server.URL, "basic")

	out, err := runCmd(t, []string{
		"-X", "POST", "/rest/api/3/issue",
		"-d", `{"summary":"hello"}`,
		"-H", "X-ExperimentalApi: opt-in",
	}, "")
	require.NoError(t, err)
	assert.Contains(t, out, `{"id":"1"}`)
}

func TestBodyImpliesPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(204)
	}))
	defer server.Close()
	setupEnv(t, server.URL, "basic")

	_, err := runCmd(t, []string{"/rest/api/3/foo", "-d", `{}`}, "")
	require.NoError(t, err)
}

func TestBodyFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "from stdin", body["summary"])
		w.WriteHeader(204)
	}))
	defer server.Close()
	setupEnv(t, server.URL, "basic")

	_, err := runCmd(t, []string{"-X", "PUT", "/rest/api/3/issue/PROJ-1", "-d", "-"}, `{"summary":"from stdin"}`)
	require.NoError(t, err)
}

func TestErrorStatusPrintsBodyAndFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
	}))
	defer server.Close()
	setupEnv(t, server.URL, "basic")

	out, err := runCmd(t, []string{"/rest/api/3/issue/NOPE-1"}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Contains(t, out, "Issue does not exist")
}

func TestListDeducesFlavorFromProfile(t *testing.T) {
	t.Run("server flavor lists v2", func(t *testing.T) {
		setupEnv(t, "https://jira.example.com", "pat")
		out, err := runCmd(t, []string{"--list", "sprint"}, "")
		require.NoError(t, err)
		assert.Contains(t, out, "/rest/api/2/")
		assert.NotContains(t, out, "/rest/api/3/")
	})

	t.Run("cloud flavor lists v3", func(t *testing.T) {
		setupEnv(t, "https://org.atlassian.net", "basic")
		out, err := runCmd(t, []string{"--list", "issue"}, "")
		require.NoError(t, err)
		assert.Contains(t, out, "/rest/api/3/")
		assert.NotContains(t, out, "/rest/api/2/")
	})
}

func TestListWorksWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_TOKEN", "")
	t.Setenv("JIRA_AUTH_TYPE", "")
	t.Setenv("JIRABOT_PROFILE", "")

	out, err := runCmd(t, []string{"--list", "GET sprint"}, "")
	require.NoError(t, err)
	assert.Contains(t, out, "sprint")
}
