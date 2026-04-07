package jira

import (
	"context"
	"fmt"
	"net/http"

	"AndersSpringborg/jira-cli/pkg/jira/cloud"
)

// DeleteIssue deletes an issue using the generated cloud client DELETE /issue/{key} endpoint.
func (c *Client) DeleteIssue(key string, cascade bool) error {
	if c.cloud == nil {
		return fmt.Errorf("cloud client not initialized")
	}

	var params *cloud.DeleteIssueParams
	if cascade {
		deleteSubtasks := cloud.DeleteIssueParamsDeleteSubtasks("true")
		params = &cloud.DeleteIssueParams{
			DeleteSubtasks: &deleteSubtasks,
		}
	}

	resp, err := c.cloud.DeleteIssueWithResponse(context.Background(), key, params)
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
