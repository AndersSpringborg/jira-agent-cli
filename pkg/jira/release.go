package jira

import (
	"context"
	"fmt"
	"net/http"
)

// Release fetches response from /project/{projectIdOrKey}/version endpoint.
func (c *Client) Release(project string) ([]*ProjectVersion, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetProjectVersionsWithResponse(
		context.Background(),
		project,
		nil, // no extra params
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

	var out []*ProjectVersion
	for _, v := range *resp.JSON200 {
		pv := &ProjectVersion{}
		if v.Id != nil {
			pv.ID = *v.Id
		}
		if v.Name != nil {
			pv.Name = *v.Name
		}
		if v.Description != nil {
			pv.Description = *v.Description
		}
		if v.Archived != nil {
			pv.Archived = *v.Archived
		}
		if v.Released != nil {
			pv.Released = *v.Released
		}
		if v.ProjectId != nil {
			pv.ProjectID = int(*v.ProjectId)
		}
		out = append(out, pv)
	}

	return out, nil
}
