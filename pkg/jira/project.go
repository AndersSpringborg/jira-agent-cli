package jira

import (
	"context"
	"fmt"
	"net/http"

	"AndersSpringborg/jira-cli/pkg/jira/cloud"
)

const (
	// ProjectTypeClassic is a classic project type.
	ProjectTypeClassic = "classic"
	// ProjectTypeNextGen is a next gen project type.
	ProjectTypeNextGen = "next-gen"
)

// Project fetches response from /project endpoint.
func (c *Client) Project() ([]*Project, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	expand := "lead"
	resp, err := c.cloud.GetAllProjectsWithResponse(
		context.Background(),
		&cloud.GetAllProjectsParams{
			Expand: &expand,
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

	var out []*Project
	for i := range *resp.JSON200 {
		p := &(*resp.JSON200)[i]
		proj := &Project{}
		if p.Key != nil {
			proj.Key = *p.Key
		}
		if p.Name != nil {
			proj.Name = *p.Name
		}
		if p.Lead != nil {
			if p.Lead.DisplayName != nil {
				proj.Lead.Name = *p.Lead.DisplayName
			}
		}
		if p.Style != nil {
			proj.Type = string(*p.Style)
		}
		out = append(out, proj)
	}

	return out, nil
}
