package jira_test

import (
	"bytes"
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

func newServerTestServer(handler http.HandlerFunc) (*httptest.Server, *jira.Client) {
	server := httptest.NewServer(handler)
	client, _ := jira.NewClient(server.URL, "", "test-token", "pat", 5)
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

func TestDebugResponseShowsRawHTMLWithoutExposingCookies(t *testing.T) {
	server, client := newServerTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Set-Cookie", "JSESSIONID=secret")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><title>Login required</title></html>"))
	})
	defer server.Close()

	var debug bytes.Buffer
	client.EnableDebug(&debug)

	_, err := client.GetMyself()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character '<'")
	assert.Contains(t, debug.String(), "--- jira debug response ---")
	assert.Contains(t, debug.String(), "GET "+server.URL+"/rest/api/2/myself")
	assert.Contains(t, debug.String(), "200 OK")
	assert.Contains(t, debug.String(), "Content-Type: text/html")
	assert.Contains(t, debug.String(), "Set-Cookie: [REDACTED]")
	assert.Contains(t, debug.String(), "<html><title>Login required</title></html>")
	assert.Contains(t, debug.String(), "--- end jira debug response ---")
	assert.NotContains(t, debug.String(), "JSESSIONID=secret")
	assert.NotContains(t, debug.String(), "test-token")
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

func TestSearchFollowsCloudPageTokens(t *testing.T) {
	requestCount := 0
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch requestCount {
		case 1:
			assert.Empty(t, r.URL.Query().Get("nextPageToken"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"nextPageToken": "page-2",
				"issues": []any{
					map[string]any{"key": "TEST-1"},
					map[string]any{"key": "TEST-2"},
				},
			})
		case 2:
			assert.Equal(t, "page-2", r.URL.Query().Get("nextPageToken"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLast": true,
				"issues": []any{
					map[string]any{"key": "TEST-3"},
				},
			})
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	})
	defer server.Close()

	data, err := client.Search("project = TEST", 0, 25)
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount)
	assert.Len(t, data["issues"], 3)
}

func TestSearchIncludesCustomFields(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		fields := r.URL.Query().Get("fields")
		assert.Contains(t, fields, "summary")
		assert.Contains(t, fields, "customfield_10145")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"total": 0, "issues": []any{}})
	})
	defer server.Close()

	_, err := client.Search("project = TEST", 0, 25, "customfield_10145")
	require.NoError(t, err)
}

func TestSearchWithChangelog(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, `issuekey IN updatedBy("abc123")`, r.URL.Query().Get("jql"))
		assert.Equal(t, "changelog", r.URL.Query().Get("expand"))
		assert.Equal(t, "50", r.URL.Query().Get("maxResults"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issues": []any{
				map[string]any{
					"key": "TEST-1",
					"changelog": map[string]any{
						"histories": []any{
							map[string]any{
								"created": "2026-06-08T10:00:00.000+0000",
								"author":  map[string]any{"accountId": "abc123"},
							},
						},
					},
				},
			},
		})
	})
	defer server.Close()

	data, err := client.SearchWithChangelog(`issuekey IN updatedBy("abc123")`, 50)
	require.NoError(t, err)

	issues := data["issues"].([]any)
	require.Len(t, issues, 1)
	iss := issues[0].(map[string]any)
	cl := iss["changelog"].(map[string]any)
	assert.Len(t, cl["histories"], 1)
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
		assert.IsType(t, map[string]any{}, fields["description"])

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

func TestServerClientUsesRESTAPI2AndWikiBody(t *testing.T) {
	server, client := newServerTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fields := body["fields"].(map[string]any)
		assert.Equal(t, "Description", fields["description"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"key": "TEST-42"})
	})
	defer server.Close()

	data, err := client.CreateIssue("TEST", "Fix bug", "Bug", "Description", "High", nil, "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "TEST-42", data["key"])
}

func TestServerClientUsesLegacySearchEndpoint(t *testing.T) {
	server, client := newServerTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/search", r.URL.Path)
		assert.Equal(t, "project = TEST", r.URL.Query().Get("jql"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"total": 0, "issues": []any{}})
	})
	defer server.Close()

	_, err := client.Search("project = TEST", 0, 25)
	require.NoError(t, err)
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

func TestEditIssue(t *testing.T) {
	t.Run("fields and update both sent", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			// Verify fields section
			fields, ok := body["fields"].(map[string]any)
			require.True(t, ok, "expected fields section")
			assert.Equal(t, "New title", fields["summary"])
			assert.Equal(t, "customval", fields["customfield_10001"])

			// Verify update section
			update, ok := body["update"].(map[string]any)
			require.True(t, ok, "expected update section")
			labelOps, ok := update["labels"].([]any)
			require.True(t, ok)
			assert.Len(t, labelOps, 1)

			w.WriteHeader(204)
		})
		defer server.Close()

		fields := map[string]any{
			"summary":           "New title",
			"customfield_10001": "customval",
		}
		update := map[string]any{
			"labels": []map[string]any{{"add": "bugfix"}},
		}
		err := client.EditIssue("TEST-1", fields, update)
		assert.NoError(t, err)
	})

	t.Run("fields only when no update ops", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			_, hasFields := body["fields"]
			assert.True(t, hasFields)
			_, hasUpdate := body["update"]
			assert.False(t, hasUpdate, "update section should be absent")

			w.WriteHeader(204)
		})
		defer server.Close()

		fields := map[string]any{"summary": "New title"}
		err := client.EditIssue("TEST-1", fields, nil)
		assert.NoError(t, err)
	})

	t.Run("update only when no fields", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			_, hasFields := body["fields"]
			assert.False(t, hasFields, "fields section should be absent")
			_, hasUpdate := body["update"]
			assert.True(t, hasUpdate)

			w.WriteHeader(204)
		})
		defer server.Close()

		update := map[string]any{
			"labels": []map[string]any{{"remove": "old-label"}},
		}
		err := client.EditIssue("TEST-1", nil, update)
		assert.NoError(t, err)
	})

	t.Run("custom fields in fields section", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			fields := body["fields"].(map[string]any)
			assert.Equal(t, "8", fields["customfield_10001"])
			assert.Equal(t, "Option A", fields["customfield_10002"])

			w.WriteHeader(204)
		})
		defer server.Close()

		fields := map[string]any{
			"customfield_10001": "8",
			"customfield_10002": "Option A",
		}
		err := client.EditIssue("TEST-1", fields, nil)
		assert.NoError(t, err)
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

func TestTransitionIssueWithUpdates(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1/transitions", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"id": "31"}, body["transition"])
		fields := body["fields"].(map[string]any)
		assert.Equal(t, map[string]any{"name": "Fixed"}, fields["resolution"])
		update := body["update"].(map[string]any)
		assert.Contains(t, update, "comment")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(204)
	})
	defer server.Close()

	err := client.TransitionIssueWithUpdates(
		"TEST-1",
		"31",
		map[string]any{"resolution": map[string]any{"name": "Fixed"}},
		map[string]any{"comment": []map[string]any{{"add": map[string]any{"body": client.CommentFieldValue("done")}}}},
	)
	require.NoError(t, err)
}

func TestAssignIssue(t *testing.T) {
	t.Run("assign by account ID", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/rest/api/3/issue/TEST-1/assignee", r.URL.Path)

			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "abc123", body["accountId"])

			w.WriteHeader(204)
		})
		defer server.Close()

		err := client.AssignIssue("TEST-1", "abc123", "", "")
		assert.NoError(t, err)
	})

	t.Run("unassign sends null accountId", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)

			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Nil(t, body["accountId"])

			w.WriteHeader(204)
		})
		defer server.Close()

		err := client.AssignIssue("TEST-1", "", "", "")
		assert.NoError(t, err)
	})

	t.Run("assign by name when accountID is empty", func(t *testing.T) {
		server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "jsmith", body["name"])
			_, hasAccountID := body["accountId"]
			assert.False(t, hasAccountID)

			w.WriteHeader(204)
		})
		defer server.Close()

		err := client.AssignIssue("TEST-1", "", "jsmith", "")
		assert.NoError(t, err)
	})
}

// TestAssignToMe verifies the two-step flow used by "jira issue assign ISSUE me":
// 1. GetMyself() returns the current user's accountId
// 2. AssignIssue() sends that accountId to the API
func TestAssignToMe(t *testing.T) {
	callCount := 0
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/api/3/myself":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accountId":   "user-abc-123",
				"displayName": "Jane Smith",
			})

		case r.Method == "PUT" && r.URL.Path == "/rest/api/3/issue/PROJ-42/assignee":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "user-abc-123", body["accountId"])
			w.WriteHeader(204)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	})
	defer server.Close()

	// Step 1: resolve "me" to an account ID
	data, err := client.GetMyself()
	require.NoError(t, err)

	accountID, ok := data["accountId"].(string)
	require.True(t, ok)
	assert.Equal(t, "user-abc-123", accountID)

	// Step 2: assign using the resolved account ID
	err = client.AssignIssue("PROJ-42", accountID, "", "")
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
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
