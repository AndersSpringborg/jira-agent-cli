package feedback_test

import (
	"testing"

	"AndersSpringborg/jira-cli/pkg/cmd/feedback"

	"github.com/stretchr/testify/assert"
)

func TestBuildIssueURL_TitleOnly(t *testing.T) {
	u := feedback.BuildIssueURL("Bug: ADF rendering broken", "")
	assert.Contains(t, u, "https://github.com/AndersSpringborg/jira-agent-cli/issues/new?")
	assert.Contains(t, u, "title=Bug")
	assert.Contains(t, u, "ADF+rendering+broken")
	assert.NotContains(t, u, "body=")
}

func TestBuildIssueURL_TitleAndBody(t *testing.T) {
	u := feedback.BuildIssueURL("Feature request", "Please add sprint burndown charts")
	assert.Contains(t, u, "title=Feature+request")
	assert.Contains(t, u, "body=Please+add+sprint+burndown+charts")
}

func TestBuildIssueURL_SpecialCharacters(t *testing.T) {
	u := feedback.BuildIssueURL("Bug: status='Done' & labels", "Body with <html> & special chars")
	// URL-encoded: & -> %26, = -> %3D, < -> %3C, > -> %3E, ' -> %27
	assert.Contains(t, u, "title=")
	assert.Contains(t, u, "body=")
	// Should not contain unencoded & in query params (other than separator)
	// The url.Values.Encode() handles this correctly
	assert.NotContains(t, u, "status=")
}

func TestBuildIssueURL_EmptyTitle(t *testing.T) {
	u := feedback.BuildIssueURL("", "")
	assert.Contains(t, u, "title=")
	// Should still produce a valid URL even with empty title
	assert.Contains(t, u, "https://github.com/AndersSpringborg/jira-agent-cli/issues/new?")
}

func TestNewCmd_RequiresTitle(t *testing.T) {
	cmd := feedback.NewCmd(nil)
	// Should fail without --title
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}
