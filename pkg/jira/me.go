package jira

import (
	"context"
	"fmt"
	"net/http"
)

// Me struct holds response from /myself endpoint.
type Me struct {
	Login    string `json:"name"`
	Name     string `json:"displayName"`
	Email    string `json:"emailAddress"`
	Timezone string `json:"timeZone"`
}

// Me fetches response from /myself endpoint.
func (c *Client) Me() (*Me, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetCurrentUserWithResponse(context.Background(), nil)
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

	u := resp.JSON200
	me := &Me{}

	if u.DisplayName != nil {
		me.Name = *u.DisplayName
	}
	if u.EmailAddress != nil {
		me.Email = *u.EmailAddress
	}
	if u.TimeZone != nil {
		me.Timezone = *u.TimeZone
	}
	// Cloud API doesn't return "name" for cloud users (it's accountId-based).
	// Use the accountId as login since that's the unique identifier.
	if u.AccountId != nil {
		me.Login = *u.AccountId
	}

	return me, nil
}
