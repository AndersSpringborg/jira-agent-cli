package apispec_test

import (
	"strings"
	"testing"

	"AndersSpringborg/jira-cli/internal/apispec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCloudReturnsV3Endpoints(t *testing.T) {
	endpoints, err := apispec.List(apispec.FlavorCloud, "")
	require.NoError(t, err)
	assert.NotEmpty(t, endpoints)

	for _, e := range endpoints {
		assert.NotEmpty(t, e.Method)
		assert.True(t, strings.HasPrefix(e.Path, "/rest/"), "path %q should start with /rest/", e.Path)
		assert.NotContains(t, e.Path, "/rest/api/2/", "cloud index should not contain v2 paths")
	}
}

func TestListServerReturnsV2Endpoints(t *testing.T) {
	endpoints, err := apispec.List(apispec.FlavorServer, "")
	require.NoError(t, err)
	assert.NotEmpty(t, endpoints)

	var hasV2, hasAgile bool
	for _, e := range endpoints {
		assert.NotContains(t, e.Path, "/rest/api/3/", "server index should not contain v3 paths")
		if strings.HasPrefix(e.Path, "/rest/api/2/") {
			hasV2 = true
		}
		if strings.HasPrefix(e.Path, "/rest/agile/1.0/") {
			hasAgile = true
		}
	}
	assert.True(t, hasV2, "server index should contain /rest/api/2 endpoints")
	assert.True(t, hasAgile, "server index should contain agile endpoints")
}

func TestListFiltersCaseInsensitively(t *testing.T) {
	endpoints, err := apispec.List(apispec.FlavorCloud, "SPRINT")
	require.NoError(t, err)
	require.NotEmpty(t, endpoints)

	for _, e := range endpoints {
		matched := strings.Contains(strings.ToLower(e.Path), "sprint") ||
			strings.Contains(strings.ToLower(e.Summary), "sprint")
		assert.True(t, matched, "endpoint %s %s (%s) should match filter", e.Method, e.Path, e.Summary)
	}
}

func TestListFilterMatchesMethod(t *testing.T) {
	endpoints, err := apispec.List(apispec.FlavorServer, "DELETE sprint")
	require.NoError(t, err)
	require.NotEmpty(t, endpoints)

	for _, e := range endpoints {
		assert.Equal(t, "DELETE", e.Method)
		assert.Contains(t, strings.ToLower(e.Path+" "+e.Summary), "sprint")
	}
}

func TestListNoMatchReturnsEmpty(t *testing.T) {
	endpoints, err := apispec.List(apispec.FlavorCloud, "definitely-not-an-endpoint-xyz")
	require.NoError(t, err)
	assert.Empty(t, endpoints)
}
