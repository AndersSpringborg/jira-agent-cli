package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"AndersSpringborg/jira-cli/pkg/adf"
	"AndersSpringborg/jira-cli/pkg/jira/cloud"
	"AndersSpringborg/jira-cli/pkg/jira/filter"
	"AndersSpringborg/jira-cli/pkg/jira/filter/issue"
	"AndersSpringborg/jira-cli/pkg/md"
)

const (
	// IssueTypeEpic is an epic issue type.
	IssueTypeEpic = "Epic"
	// IssueTypeSubTask is a sub-task issue type.
	IssueTypeSubTask = "Sub-task"
	// AssigneeNone is an empty assignee.
	AssigneeNone = "none"
	// AssigneeDefault is a default assignee.
	AssigneeDefault = "default"
)

// GetIssue fetches issue details using the generated cloud client GET /issue/{key} endpoint.
func (c *Client) GetIssue(key string, opts ...filter.Filter) (*Issue, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetIssueWithResponse(
		context.Background(),
		key,
		nil, // default params: returns all fields
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

	iss := convertIssueBean(resp.JSON200)

	iss.Fields.Description = ifaceToADF(iss.Fields.Description)

	total := iss.Fields.Comment.Total
	limit := filter.Collection(opts).GetInt(issue.KeyIssueNumComments)
	if limit > total {
		limit = total
	}
	for i := total - 1; i >= total-limit; i-- {
		body := iss.Fields.Comment.Comments[i].Body
		iss.Fields.Comment.Comments[i].Body = ifaceToADF(body)
	}
	return iss, nil
}

// GetIssueV2 fetches issue details using v2 version of Jira GET /issue/{key} endpoint.
func (c *Client) GetIssueV2(key string, _ ...filter.Filter) (*Issue, error) {
	return c.getIssue(key, apiVersion2)
}

func (c *Client) getIssue(key, ver string) (*Issue, error) {
	rawOut, err := c.getIssueRaw(key, ver)
	if err != nil {
		return nil, err
	}

	var iss Issue
	err = json.Unmarshal([]byte(rawOut), &iss)
	if err != nil {
		return nil, err
	}
	return &iss, nil
}

// GetIssueRaw fetches issue details using the generated cloud client but returns the raw API response body string.
func (c *Client) GetIssueRaw(key string) (string, error) {
	if c.cloud == nil {
		return "", fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetIssueWithResponse(
		context.Background(),
		key,
		nil,
	)
	if err != nil {
		return "", err
	}
	if resp.HTTPResponse == nil {
		return "", ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusOK {
		return "", parseCloudError(resp.Body, resp.HTTPResponse)
	}

	return string(resp.Body), nil
}

// GetIssueV2Raw fetches issue details same as GetIssueV2 but returns the raw API response body string.
func (c *Client) GetIssueV2Raw(key string) (string, error) {
	return c.getIssueRaw(key, apiVersion2)
}

func (c *Client) getIssueRaw(key, ver string) (string, error) {
	path := fmt.Sprintf("/issue/%s", key)

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
		return "", err
	}
	if res == nil {
		return "", ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", formatUnexpectedResponse(res)
	}

	var b strings.Builder
	_, err = io.Copy(&b, res.Body)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// AssignIssue assigns issue to the user using the generated cloud client PUT /issue/{key}/assignee endpoint.
func (c *Client) AssignIssue(key, assignee string) error {
	if c.cloud == nil {
		return fmt.Errorf("cloud client not initialized")
	}

	var accountID *string
	switch assignee {
	case AssigneeNone:
		v := "-1"
		accountID = &v
	case AssigneeDefault:
		accountID = nil
	default:
		accountID = &assignee
	}

	body := cloud.AssignIssueJSONRequestBody{
		AccountId: accountID,
	}

	resp, err := c.cloud.AssignIssueWithResponse(context.Background(), key, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse == nil {
		return ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusNoContent {
		return parseCloudError(resp.Body, resp.HTTPResponse)
	}
	return nil
}

// AssignIssueV2 assigns issue to the user using v2 version of the PUT /issue/{key}/assignee endpoint.
func (c *Client) AssignIssueV2(key, assignee string) error {
	return c.assignIssue(key, assignee, apiVersion2)
}

func (c *Client) assignIssue(key, assignee, ver string) error {
	path := fmt.Sprintf("/issue/%s/assignee", key)

	aid := new(string)
	switch assignee {
	case AssigneeNone:
		*aid = "-1"
	case AssigneeDefault:
		aid = nil
	default:
		*aid = assignee
	}

	var (
		res  *http.Response
		err  error
		body []byte
	)

	switch ver {
	case apiVersion2:
		type assignRequest struct {
			Name *string `json:"name"`
		}

		body, err = json.Marshal(assignRequest{Name: aid})
		if err != nil {
			return err
		}
		res, err = c.PutV2(context.Background(), path, body, Header{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
	default:
		type assignRequest struct {
			AccountID *string `json:"accountId"`
		}

		body, err = json.Marshal(assignRequest{AccountID: aid})
		if err != nil {
			return err
		}
		res, err = c.Put(context.Background(), path, body, Header{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
	}

	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent {
		return formatUnexpectedResponse(res)
	}
	return nil
}

// GetIssueLinkTypes fetches issue link types using the generated cloud client GET /issueLinkType endpoint.
func (c *Client) GetIssueLinkTypes() ([]*IssueLinkType, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetIssueLinkTypesWithResponse(context.Background())
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

	var out []*IssueLinkType
	if resp.JSON200.IssueLinkTypes != nil {
		for _, lt := range *resp.JSON200.IssueLinkTypes {
			ilt := &IssueLinkType{}
			if lt.Id != nil {
				ilt.ID = *lt.Id
			}
			if lt.Name != nil {
				ilt.Name = *lt.Name
			}
			if lt.Inward != nil {
				ilt.Inward = *lt.Inward
			}
			if lt.Outward != nil {
				ilt.Outward = *lt.Outward
			}
			out = append(out, ilt)
		}
	}

	return out, nil
}

// LinkIssue connects issues to the given link type using the generated cloud client POST /issueLink endpoint.
func (c *Client) LinkIssue(inwardIssue, outwardIssue, linkType string) error {
	if c.cloud == nil {
		return fmt.Errorf("cloud client not initialized")
	}

	body := cloud.LinkIssuesJSONRequestBody{
		InwardIssue: cloud.LinkedIssue{
			Key: &inwardIssue,
		},
		OutwardIssue: cloud.LinkedIssue{
			Key: &outwardIssue,
		},
		Type: cloud.IssueLinkType{
			Name: &linkType,
		},
	}

	resp, err := c.cloud.LinkIssuesWithResponse(context.Background(), body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse == nil {
		return ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusCreated {
		return parseCloudError(resp.Body, resp.HTTPResponse)
	}
	return nil
}

// UnlinkIssue disconnects two issues using the generated cloud client DELETE /issueLink/{linkId} endpoint.
func (c *Client) UnlinkIssue(linkID string) error {
	if c.cloud == nil {
		return fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.DeleteIssueLinkWithResponse(context.Background(), linkID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse == nil {
		return ErrEmptyResponse
	}
	// The API returns 200 on successful delete for issue links.
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return parseCloudError(resp.Body, resp.HTTPResponse)
	}
	return nil
}

// GetLinkID gets linkID between two issues.
func (c *Client) GetLinkID(inwardIssue, outwardIssue string) (string, error) {
	i, err := c.GetIssueV2(inwardIssue)
	if err != nil {
		return "", err
	}

	for _, link := range i.Fields.IssueLinks {
		if link.InwardIssue != nil && link.InwardIssue.Key == outwardIssue {
			return link.ID, nil
		}

		if link.OutwardIssue != nil && link.OutwardIssue.Key == outwardIssue {
			return link.ID, nil
		}
	}
	return "", fmt.Errorf("no link found between provided issues")
}

type issueCommentPropertyValue struct {
	Internal bool `json:"internal"`
}

type issueCommentProperty struct {
	Key   string                    `json:"key"`
	Value issueCommentPropertyValue `json:"value"`
}
type issueCommentRequest struct {
	Body       string                 `json:"body"`
	Properties []issueCommentProperty `json:"properties"`
}

// AddIssueComment adds comment to an issue using POST /issue/{key}/comment endpoint.
// Note: Uses v2 API for Jira wiki markup body format.
func (c *Client) AddIssueComment(key, comment string, internal bool) error {
	body, err := json.Marshal(&issueCommentRequest{Body: md.ToJiraMD(comment), Properties: []issueCommentProperty{{Key: "sd.public.comment", Value: issueCommentPropertyValue{Internal: internal}}}})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/comment", key)
	res, err := c.PostV2(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
}

type issueWorklogRequest struct {
	Started   string `json:"started,omitempty"`
	TimeSpent string `json:"timeSpent"`
	Comment   string `json:"comment"`
}

// AddIssueWorklog adds worklog to an issue using POST /issue/{key}/worklog endpoint.
// Leave param `started` empty to use the server's current datetime as start date.
// Note: Uses v2 API for Jira wiki markup comment format.
func (c *Client) AddIssueWorklog(key, started, timeSpent, comment, newEstimate string) error {
	worklogReq := issueWorklogRequest{
		TimeSpent: timeSpent,
		Comment:   md.ToJiraMD(comment),
	}
	if started != "" {
		worklogReq.Started = started
	}
	body, err := json.Marshal(&worklogReq)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/worklog", key)
	if newEstimate != "" {
		path = fmt.Sprintf("%s?adjustEstimate=new&newEstimate=%s", path, newEstimate)
	}
	res, err := c.PostV2(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
}

// GetField gets all fields configured for a Jira instance using the generated cloud client GET /field endpoint.
func (c *Client) GetField() ([]*Field, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetFieldsWithResponse(context.Background())
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

	var out []*Field
	for _, f := range *resp.JSON200 {
		field := &Field{}
		if f.Id != nil {
			field.ID = *f.Id
		}
		if f.Name != nil {
			field.Name = *f.Name
		}
		if f.Custom != nil {
			field.Custom = *f.Custom
		}
		if f.Schema != nil {
			if f.Schema.Type != nil {
				field.Schema.DataType = *f.Schema.Type
			}
			if f.Schema.Items != nil {
				field.Schema.Items = *f.Schema.Items
			}
			if f.Schema.CustomId != nil {
				field.Schema.FieldID = int(*f.Schema.CustomId)
			}
		}
		out = append(out, field)
	}

	return out, nil
}

func ifaceToADF(v interface{}) *adf.ADF {
	if v == nil {
		return nil
	}

	var doc *adf.ADF

	js, err := json.Marshal(v)
	if err != nil {
		return nil // ignore invalid data
	}
	if err = json.Unmarshal(js, &doc); err != nil {
		return nil // ignore invalid data
	}

	return doc
}

type remotelinkRequest struct {
	RemoteObject struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"object"`
}

// RemoteLinkIssue adds a remote link to an issue using POST /issue/{issueId}/remotelink endpoint.
// Note: Uses v2 API since the generated client uses v3 which has a different body format.
func (c *Client) RemoteLinkIssue(issueID, title, url string) error {
	body, err := json.Marshal(remotelinkRequest{
		RemoteObject: struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}{Title: title, URL: url},
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/remotelink", issueID)

	res, err := c.PostV2(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
}

// WatchIssue adds user as a watcher using the generated cloud client POST /issue/{key}/watchers endpoint.
func (c *Client) WatchIssue(key, watcher string) error {
	if c.cloud == nil {
		return fmt.Errorf("cloud client not initialized")
	}

	body := cloud.AddWatcherJSONRequestBody(watcher)
	resp, err := c.cloud.AddWatcherWithResponse(context.Background(), key, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse == nil {
		return ErrEmptyResponse
	}
	if resp.StatusCode() != http.StatusNoContent {
		return parseCloudError(resp.Body, resp.HTTPResponse)
	}
	return nil
}

// WatchIssueV2 adds user as a watcher using v2 version of the POST /issue/{key}/watchers endpoint.
func (c *Client) WatchIssueV2(key, watcher string) error {
	return c.watchIssue(key, watcher, apiVersion2)
}

func (c *Client) watchIssue(key, watcher, ver string) error {
	path := fmt.Sprintf("/issue/%s/watchers", key)

	var (
		res  *http.Response
		err  error
		body []byte
	)

	body, err = json.Marshal(watcher)
	if err != nil {
		return err
	}

	header := Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	switch ver {
	case apiVersion2:
		res, err = c.PostV2(context.Background(), path, body, header)
	default:
		res, err = c.Post(context.Background(), path, body, header)
	}

	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent {
		return formatUnexpectedResponse(res)
	}
	return nil
}
