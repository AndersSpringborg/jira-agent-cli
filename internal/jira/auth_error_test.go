package jira_test

import (
	"net/http"
	"testing"

	"AndersSpringborg/jira-cli/internal/jira"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Jira Server/DC ignores an expired PAT and serves the request anonymously,
// so the failure surfaces as a misleading 400 about invisible fields.
func TestSearchWithExpiredTokenReportsAuthenticationFailure(t *testing.T) {
	server, client := newServerTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ausername", "anonymous")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errorMessages":["The value 'CAI' does not exist for the field 'project'.","Field 'Epic Link' does not exist or this field cannot be viewed by anonymous users."],"errors":{}}`))
	})
	defer server.Close()

	_, err := client.Search("project = CAI", 0, 1)

	require.ErrorIs(t, err, jira.ErrAuthentication)
	assert.Contains(t, err.Error(), "expired")
	assert.Contains(t, err.Error(), "jira auth login")
}

func TestUnauthorizedResponseReportsAuthenticationFailure(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Client must be authenticated to access this resource."))
	})
	defer server.Close()

	_, err := client.GetMyself()

	require.ErrorIs(t, err, jira.ErrAuthentication)
}

// An anonymous request can also succeed with an empty result, which would
// silently hide every issue the token used to see.
func TestSuccessfulAnonymousResponseReportsAuthenticationFailure(t *testing.T) {
	server, client := newServerTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ausername", "anonymous")
		_, _ = w.Write([]byte(`{"issues":[],"total":0}`))
	})
	defer server.Close()

	_, err := client.Search("project = PUBLIC", 0, 10)

	require.ErrorIs(t, err, jira.ErrAuthentication)
}
