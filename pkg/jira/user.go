package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"AndersSpringborg/jira-cli/pkg/jira/cloud"
)

// ErrInvalidSearchOption denotes invalid search option was given.
var ErrInvalidSearchOption = fmt.Errorf("invalid search option")

// UserSearchOptions holds options to search for user.
type UserSearchOptions struct {
	Project    string
	Query      string
	Username   string
	AccountID  string
	StartAt    int
	MaxResults int
}

// UserSearch search for user details using the generated cloud client
// GET /user/assignable/search endpoint.
func (c *Client) UserSearch(opt *UserSearchOptions) ([]*User, error) {
	if opt == nil {
		return nil, ErrInvalidSearchOption
	}
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	params := &cloud.FindAssignableUsersParams{}

	if opt.Project != "" {
		params.Project = &opt.Project
	}
	if opt.Query != "" {
		params.Query = &opt.Query
	}
	if opt.AccountID != "" {
		params.AccountId = &opt.AccountID
	}
	if opt.StartAt != 0 {
		startAt := int32(opt.StartAt)
		params.StartAt = &startAt
	}
	if opt.MaxResults != 0 {
		maxResults := int32(opt.MaxResults)
		params.MaxResults = &maxResults
	}

	// Validate that at least one search option is set.
	if opt.Project == "" && opt.Query == "" && opt.AccountID == "" {
		return nil, ErrInvalidSearchOption
	}

	resp, err := c.cloud.FindAssignableUsersWithResponse(context.Background(), params)
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

	return convertUsers(*resp.JSON200), nil
}

// UserSearchV2 search for user details using v2 version of the GET /user/assignable/search endpoint.
func (c *Client) UserSearchV2(opt *UserSearchOptions) ([]*User, error) {
	// The `username` query param is deprecated since Jira API v2 and is not available in v3.
	// Since the` query` parameter doesn't seem to return expected results, we will use the
	// `username` param in call to v2. Chances are the `query` param may stop working in
	// later v2 updates so we might have to revisit this in the future. Note that the
	// `username` param is not as flexible as the `query` param available in v3.
	//
	// See https://github.com/ankitpokhrel/jira-cli/issues/198
	if opt.Query != "" && opt.Username == "" {
		opt.Username = opt.Query
		opt.Query = ""
	}
	return c.userSearch(opt, apiVersion2)
}

func (c *Client) userSearch(opt *UserSearchOptions, ver string) ([]*User, error) {
	if opt == nil {
		return nil, ErrInvalidSearchOption
	}

	var (
		opts []string
		res  *http.Response
		err  error
	)

	if opt.Project != "" {
		opts = append(opts, fmt.Sprintf("project=%s", opt.Project))
	}
	if opt.Query != "" {
		opts = append(opts, fmt.Sprintf("query=%s", url.QueryEscape(opt.Query)))
	}
	if opt.Username != "" {
		opts = append(opts, fmt.Sprintf("username=%s", url.QueryEscape(opt.Username)))
	}
	if opt.AccountID != "" {
		opts = append(opts, fmt.Sprintf("accountId=%s", url.QueryEscape(opt.AccountID)))
	}
	if opt.StartAt != 0 {
		opts = append(opts, fmt.Sprintf("startAt=%d", opt.StartAt))
	}
	if opt.MaxResults != 0 {
		opts = append(opts, fmt.Sprintf("maxResults=%d", opt.MaxResults))
	}
	if len(opts) == 0 {
		return nil, ErrInvalidSearchOption
	}

	path := fmt.Sprintf("%s?%s", "/user/assignable/search", strings.Join(opts, "&"))

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

	var out []*User
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// convertUsers maps a slice of generated cloud User types to our domain User type.
func convertUsers(users []cloud.User) []*User {
	var out []*User
	for i := range users {
		u := &users[i]
		user := &User{}
		if u.AccountId != nil {
			user.AccountID = *u.AccountId
		}
		if u.DisplayName != nil {
			user.DisplayName = *u.DisplayName
		}
		if u.EmailAddress != nil {
			user.Email = *u.EmailAddress
		}
		if u.Active != nil {
			user.Active = *u.Active
		}
		out = append(out, user)
	}
	return out
}
