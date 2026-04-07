package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"AndersSpringborg/jira-cli/pkg/jira/cloud"
)

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	IsLast        bool     `json:"isLast"`
	NextPageToken string   `json:"nextPageToken"`
	Issues        []*Issue `json:"issues"`
}

// Search searches for issues using v3 version of the Jira GET /search/jql endpoint.
func (c *Client) Search(jql string, limit uint) (*SearchResult, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	maxResults := int32(limit)
	fields := []string{"*all"}
	resp, err := c.cloud.SearchAndReconsileIssuesUsingJqlWithResponse(
		context.Background(),
		&cloud.SearchAndReconsileIssuesUsingJqlParams{
			Jql:        &jql,
			MaxResults: &maxResults,
			Fields:     &fields,
		},
	)
	if err != nil {
		return nil, err
	}
	if resp.HTTPResponse == nil {
		return nil, ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, parseCloudError(resp.Body, resp.HTTPResponse)
	}
	if resp.JSON200 == nil {
		return nil, ErrEmptyResponse
	}

	return convertSearchResult(resp.JSON200), nil
}

// SearchV2 searches an issues using v2 version of the Jira GET /search endpoint.
func (c *Client) SearchV2(jql string, from, limit uint) (*SearchResult, error) {
	path := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d", url.QueryEscape(jql), from, limit)
	return c.search(path, apiVersion2)
}

func (c *Client) search(path, ver string) (*SearchResult, error) {
	var (
		res *http.Response
		err error
	)

	switch ver {
	case apiVersion2:
		res, err = c.GetV2(context.Background(), path, nil)
	default:
		res, err = c.Get(context.Background(), path, nil)
	}

	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out SearchResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

// convertSearchResult maps the generated cloud type to our domain type.
func convertSearchResult(sr *cloud.SearchAndReconcileResults) *SearchResult {
	result := &SearchResult{}

	if sr.IsLast != nil {
		result.IsLast = *sr.IsLast
	}
	if sr.NextPageToken != nil {
		result.NextPageToken = *sr.NextPageToken
	}
	if sr.Issues != nil {
		for _, iss := range *sr.Issues {
			result.Issues = append(result.Issues, convertIssueBean(&iss))
		}
	}

	return result
}

// convertIssueBean maps a generated IssueBean to our domain Issue type.
// IssueBean has Fields as map[string]interface{} so we re-marshal through JSON.
func convertIssueBean(ib *cloud.IssueBean) *Issue {
	issue := &Issue{}

	if ib.Key != nil {
		issue.Key = *ib.Key
	}

	if ib.Fields != nil {
		// Marshal the fields map back to JSON and decode into our IssueFields struct.
		data, err := json.Marshal(*ib.Fields)
		if err == nil {
			_ = json.Unmarshal(data, &issue.Fields)
		}
	}

	return issue
}

// parseCloudError constructs an ErrUnexpectedResponse from the generated client's response body.
func parseCloudError(body []byte, resp *http.Response) *ErrUnexpectedResponse {
	var b Errors
	_ = json.Unmarshal(body, &b)

	return &ErrUnexpectedResponse{
		Body:       b,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
}
