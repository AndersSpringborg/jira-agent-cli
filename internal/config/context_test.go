package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildJQLIncludesAllIssuesUnderEpic(t *testing.T) {
	jql := BuildJQL(&Context{Project: "CER", Epic: "CER-33"})

	assert.Equal(t, `project = "CER" AND ("Epic Link" = "CER-33" OR parent = "CER-33")`, jql)
}
