package jira

import (
	"fmt"

	"AndersSpringborg/jira-cli/pkg/md"
)

// Strategy describes the Jira REST API dialect used by a client.
// Jira Cloud uses REST API v3 and ADF document bodies. Jira Server/Data Center
// uses REST API v2 and wiki/plain string bodies.
type Strategy interface {
	APIPath(resource string) string
	SearchPath() string
	TextBody(text string) any
	CommentBody(text string) any
	AssignBody(accountID, name string) map[string]any
	AssigneeField(accountID, name string) map[string]any
	UserSearchPath(query string) string
	UserPath(accountID string) string
}

type cloudStrategy struct{}

func (cloudStrategy) APIPath(resource string) string { return "/rest/api/3/" + resource }
func (cloudStrategy) SearchPath() string             { return "/rest/api/3/search/jql" }
func (cloudStrategy) TextBody(text string) any       { return markdownToADF(text) }
func (cloudStrategy) CommentBody(text string) any    { return markdownToADF(text) }
func (cloudStrategy) AssignBody(accountID, name string) map[string]any {
	body := map[string]any{}
	switch {
	case accountID != "":
		body["accountId"] = accountID
	case name != "":
		body["name"] = name
	default:
		body["accountId"] = nil
	}
	return body
}
func (cloudStrategy) AssigneeField(accountID, name string) map[string]any {
	if accountID != "" {
		return map[string]any{"accountId": accountID}
	}
	return map[string]any{"accountId": name}
}
func (cloudStrategy) UserSearchPath(query string) string {
	return fmt.Sprintf("/rest/api/3/user/search?query=%s", urlEncode(query))
}
func (cloudStrategy) UserPath(accountID string) string {
	return fmt.Sprintf("/rest/api/3/user?accountId=%s", urlEncode(accountID))
}

type serverStrategy struct{}

func (serverStrategy) APIPath(resource string) string { return "/rest/api/2/" + resource }
func (serverStrategy) SearchPath() string             { return "/rest/api/2/search" }
func (serverStrategy) TextBody(text string) any       { return md.ToJiraMD(text) }
func (serverStrategy) CommentBody(text string) any    { return md.ToJiraMD(text) }
func (serverStrategy) AssignBody(accountID, name string) map[string]any {
	body := map[string]any{}
	switch {
	case name != "":
		body["name"] = name
	case accountID != "":
		body["name"] = accountID
	default:
		body["name"] = nil
	}
	return body
}
func (serverStrategy) AssigneeField(accountID, name string) map[string]any {
	if name != "" {
		return map[string]any{"name": name}
	}
	return map[string]any{"name": accountID}
}
func (serverStrategy) UserSearchPath(query string) string {
	return fmt.Sprintf("/rest/api/2/user/search?username=%s", urlEncode(query))
}
func (serverStrategy) UserPath(accountID string) string {
	return fmt.Sprintf("/rest/api/2/user?username=%s", urlEncode(accountID))
}

func selectStrategy(_ string, authType string) Strategy {
	if authType == "pat" || authType == "bearer" {
		return serverStrategy{}
	}
	return cloudStrategy{}
}
