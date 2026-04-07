package jira

import (
	"context"
	"fmt"
	"net/http"
)

// ServerInfo struct holds response from /serverInfo endpoint.
type ServerInfo struct {
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
	DeploymentType string `json:"deploymentType"`
	BuildNumber    int    `json:"buildNumber"`
	DefaultLocale  struct {
		Locale string `json:"locale"`
	} `json:"defaultLocale"`
}

// ServerInfo fetches response from /serverInfo endpoint.
func (c *Client) ServerInfo() (*ServerInfo, error) {
	if c.cloud == nil {
		return nil, fmt.Errorf("cloud client not initialized")
	}

	resp, err := c.cloud.GetServerInfoWithResponse(context.Background())
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

	si := resp.JSON200
	info := &ServerInfo{}

	if si.Version != nil {
		info.Version = *si.Version
	}
	if si.DeploymentType != nil {
		info.DeploymentType = *si.DeploymentType
	}
	if si.BuildNumber != nil {
		info.BuildNumber = int(*si.BuildNumber)
	}
	if si.VersionNumbers != nil {
		for _, v := range *si.VersionNumbers {
			info.VersionNumbers = append(info.VersionNumbers, int(v))
		}
	}

	return info, nil
}
