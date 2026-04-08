package jira_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"AndersSpringborg/jira-cli/internal/jira"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *jira.Client) {
	server := httptest.NewServer(handler)
	client, _ := jira.NewClient(server.URL, "test@example.com", "test-token", "basic", 5)
	return server, client
}

func TestNewClient(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c, err := jira.NewClient("https://jira.example.com", "user@example.com", "token", "basic", 10)
		require.NoError(t, err)
		assert.Equal(t, "https://jira.example.com", c.BaseURL)
	})

	t.Run("trims trailing slash", func(t *testing.T) {
		c, err := jira.NewClient("https://jira.example.com/", "user@example.com", "token", "basic", 10)
		require.NoError(t, err)
		assert.Equal(t, "https://jira.example.com", c.BaseURL)
	})

	t.Run("empty base URL fails", func(t *testing.T) {
		_, err := jira.NewClient("", "user@example.com", "token", "basic", 10)
		assert.Error(t, err)
	})
}

func TestGetMyself(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/myself", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"displayName":  "Jane Smith",
			"emailAddress": "jane@example.com",
			"accountId":    "abc123",
		})
	})
	defer server.Close()

	data, err := client.GetMyself()
	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", data["displayName"])
	assert.Equal(t, "jane@example.com", data["emailAddress"])
}

func TestGetIssue(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "TEST-1",
			"fields": map[string]any{
				"summary": "Test issue",
				"status":  map[string]any{"name": "To Do"},
			},
		})
	})
	defer server.Close()

	data, err := client.GetIssue("TEST-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "TEST-1", data["key"])

	fields := data["fields"].(map[string]any)
	assert.Equal(t, "Test issue", fields["summary"])
}

func TestGetIssue_WithFields(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "summary,status", r.URL.Query().Get("fields"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"key": "TEST-1"})
	})
	defer server.Close()

	_, err := client.GetIssue("TEST-1", []string{"summary", "status"})
	require.NoError(t, err)
}

func TestSearch(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "project = TEST", r.URL.Query().Get("jql"))
		assert.Equal(t, "0", r.URL.Query().Get("startAt"))
		assert.Equal(t, "25", r.URL.Query().Get("maxResults"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"issues": []any{
				map[string]any{
					"key": "TEST-1",
					"fields": map[string]any{
						"summary": "Found issue",
					},
				},
			},
		})
	})
	defer server.Close()

	data, err := client.Search("project = TEST", 0, 25)
	require.NoError(t, err)
	assert.Equal(t, float64(1), data["total"])

	issues := data["issues"].([]any)
	assert.Len(t, issues, 1)
}

func TestCreateIssue(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fields := body["fields"].(map[string]any)
		project := fields["project"].(map[string]any)
		assert.Equal(t, "TEST", project["key"])
		assert.Equal(t, "Fix bug", fields["summary"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":  "10001",
			"key": "TEST-42",
		})
	})
	defer server.Close()

	data, err := client.CreateIssue("TEST", "Fix bug", "Bug", "Description", "High", nil, "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "TEST-42", data["key"])
}

func TestDeleteIssue(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	})
	defer server.Close()

	err := client.DeleteIssue("TEST-1")
	assert.NoError(t, err)
}

func TestListBoards(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/rest/agile/1.0/board")
		assert.Equal(t, "50", r.URL.Query().Get("maxResults"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{
				map[string]any{"id": float64(1), "name": "Board A", "type": "scrum"},
				map[string]any{"id": float64(2), "name": "Board B", "type": "kanban"},
			},
		})
	})
	defer server.Close()

	boards, err := client.ListBoards("", 50, "")
	require.NoError(t, err)
	assert.Len(t, boards, 2)
	assert.Equal(t, "Board A", boards[0]["name"])
}

func TestListSprints(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/rest/agile/1.0/board/42/sprint")
		assert.Equal(t, "active", r.URL.Query().Get("state"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{
				map[string]any{
					"id":    float64(10),
					"name":  "Sprint 5",
					"state": "active",
				},
			},
		})
	})
	defer server.Close()

	sprints, err := client.ListSprints(42, "active")
	require.NoError(t, err)
	assert.Len(t, sprints, 1)
	assert.Equal(t, "Sprint 5", sprints[0]["name"])
}

func TestGetIssueTransitions(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1/transitions", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"transitions": []any{
				map[string]any{"id": "11", "name": "To Do"},
				map[string]any{"id": "21", "name": "In Progress"},
				map[string]any{"id": "31", "name": "Done"},
			},
		})
	})
	defer server.Close()

	transitions, err := client.GetIssueTransitions("TEST-1")
	require.NoError(t, err)
	assert.Len(t, transitions, 3)
	assert.Equal(t, "In Progress", transitions[1]["name"])
}

func TestListProjects(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"key": "PROJ", "name": "My Project"},
			{"key": "TEST", "name": "Test Project"},
		})
	})
	defer server.Close()

	projects, err := client.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "PROJ", projects[0]["key"])
}

func TestListUsers(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/user/search", r.URL.Path)
		assert.Equal(t, "jane", r.URL.Query().Get("query"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"displayName": "Jane Smith", "accountId": "abc123"},
		})
	})
	defer server.Close()

	users, err := client.ListUsers("jane")
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Jane Smith", users[0]["displayName"])
}

func TestErrorHandling(t *testing.T) {
	t.Run("jira error format", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errorMessages": []string{"Issue does not exist"},
				"errors":        map[string]string{},
			})
		})
		defer server.Close()

		_, err := client.GetIssue("NOPE-1", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Issue does not exist")
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("non-json error", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte("Internal Server Error"))
		})
		defer server.Close()

		_, err := client.GetMyself()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestBasicAuth(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "test@example.com", user)
		assert.Equal(t, "test-token", pass)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	defer server.Close()

	_, err := client.GetMyself()
	assert.NoError(t, err)
}

func TestBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-pat", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, _ := jira.NewClient(server.URL, "", "my-pat", "pat", 5)
	_, err := client.GetMyself()
	assert.NoError(t, err)
}
