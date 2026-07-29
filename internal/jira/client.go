// Package jira provides a thin HTTP client for the Jira REST API.
//
// All methods return untyped data (map[string]any or []map[string]any)
// making the responses easy to pass directly to display drivers
// (JSON, markdown) without intermediate type conversion.
//
// This is intentional: an AI-first CLI benefits from passing through
// the API's JSON structure unchanged, so that jq pipelines and LLM
// tool parsers get the full Jira response fidelity.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a Jira REST API client that returns untyped JSON data.
type Client struct {
	BaseURL    string
	httpClient *http.Client
	email      string
	token      string
	authType   string // "basic" or "pat"/"bearer"
	strategy   Strategy
	debug      io.Writer
}

// NewClient creates a new Jira client.
func NewClient(baseURL, email, token, authType string, timeout float64) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("jira: base URL is required")
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if timeout <= 0 {
		timeout = 15
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: time.Duration(timeout) * time.Second,
		}).DialContext,
	}

	return &Client{
		BaseURL:  baseURL,
		email:    email,
		token:    token,
		authType: authType,
		strategy: selectStrategy(baseURL, authType),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(timeout) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				if len(via) > 0 && !sameOrigin(req.URL, via[0].URL) {
					req.Header.Del("Authorization")
					req.Header.Del("Cookie")
					req.Header.Del("Proxy-Authorization")
				}
				return nil
			},
		},
	}, nil
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

// EnableDebug writes raw HTTP responses to w while preserving response bodies
// for normal decoding. Authentication request headers are never included.
func (c *Client) EnableDebug(w io.Writer) {
	c.debug = w
}

// --- User / Auth ---

// GetMyself returns the currently authenticated user.
func (c *Client) GetMyself() (map[string]any, error) {
	return c.getJSON(c.strategy.APIPath("myself"))
}

// --- Issues ---

// GetIssue fetches a single issue by key.
// fields optionally limits which fields are returned.
func (c *Client) GetIssue(key string, fields []string) (map[string]any, error) {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s", key))
	if len(fields) > 0 {
		path += "?fields=" + strings.Join(fields, ",")
	}
	return c.getJSON(path)
}

// DownloadAttachment downloads attachment content from a URL returned by Jira.
// Credentials are sent only when the content URL has the same origin as the
// configured Jira instance. Go's HTTP client also strips credentials when a
// response redirects to an unrelated host.
func (c *Client) DownloadAttachment(contentURL string) (*http.Response, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Jira base URL: %w", err)
	}
	content, err := url.Parse(contentURL)
	if err != nil {
		return nil, fmt.Errorf("parse attachment URL: %w", err)
	}

	if content.IsAbs() {
		if !strings.EqualFold(content.Scheme, base.Scheme) || !strings.EqualFold(content.Host, base.Host) {
			return nil, fmt.Errorf("attachment URL is outside Jira origin: %s", contentURL)
		}
	} else if !strings.HasPrefix(content.Path, "/") {
		return nil, fmt.Errorf("attachment URL must be absolute or root-relative: %s", contentURL)
	}

	path := content.EscapedPath()
	basePath := strings.TrimSuffix(base.EscapedPath(), "/")
	if basePath != "" {
		switch {
		case path == basePath:
			path = "/"
		case strings.HasPrefix(path, basePath+"/"):
			path = strings.TrimPrefix(path, basePath)
		case content.IsAbs():
			return nil, fmt.Errorf("attachment URL is outside Jira base path: %s", contentURL)
		}
	}
	if content.RawQuery != "" {
		path += "?" + content.RawQuery
	}
	resp, err := c.do("GET", path, nil, map[string]string{"Accept": "*/*"})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close() //nolint:errcheck // preserve the response error
		return nil, c.formatError(resp)
	}
	return resp, nil
}

// CreateIssue creates a new issue and returns the created issue data.
func (c *Client) CreateIssue(project, summary, issueType, description, priority string, labels []string, parent string, components, fixVersions []string) (map[string]any, error) {
	fields := map[string]any{
		"project":   map[string]any{"key": project},
		"summary":   summary,
		"issuetype": map[string]any{"name": issueType},
	}
	if description != "" {
		fields["description"] = c.strategy.TextBody(description)
	}
	if priority != "" {
		fields["priority"] = map[string]any{"name": priority}
	}
	if len(labels) > 0 {
		fields["labels"] = labels
	}
	if parent != "" {
		fields["parent"] = map[string]any{"key": parent}
	}
	if len(components) > 0 {
		comps := make([]map[string]any, len(components))
		for i, name := range components {
			comps[i] = map[string]any{"name": name}
		}
		fields["components"] = comps
	}
	if len(fixVersions) > 0 {
		versions := make([]map[string]any, len(fixVersions))
		for i, name := range fixVersions {
			versions[i] = map[string]any{"name": name}
		}
		fields["fixVersions"] = versions
	}

	body := map[string]any{
		"fields": fields,
	}
	return c.postJSON(c.strategy.APIPath("issue"), body)
}

// UpdateIssue updates an existing issue's fields.
func (c *Client) UpdateIssue(key string, fields map[string]any) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s", key))
	if desc, ok := fields["description"].(string); ok && desc != "" {
		fields["description"] = c.strategy.TextBody(desc)
	}
	body := map[string]any{"fields": fields}
	return c.putNoContent(path, body)
}

// EditIssue updates an issue using both the "fields" and "update" sections
// of the Jira edit API. Use "fields" for simple set operations (summary,
// description, priority, custom fields) and "update" for array operations
// (add/remove on labels, components, fixVersions).
func (c *Client) EditIssue(key string, fields, update map[string]any) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s", key))
	if desc, ok := fields["description"].(string); ok && desc != "" {
		fields["description"] = c.strategy.TextBody(desc)
	}
	body := map[string]any{}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	if len(update) > 0 {
		body["update"] = update
	}
	return c.putNoContent(path, body)
}

// DeleteIssue deletes an issue by key.
func (c *Client) DeleteIssue(key string) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s", key))
	return c.delete(path)
}

// CloneIssue clones an issue with optional field overrides.
func (c *Client) CloneIssue(key string, overrides map[string]any) (map[string]any, error) {
	// First get the original issue
	original, err := c.GetIssue(key, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch issue to clone: %w", err)
	}

	fields, _ := original["fields"].(map[string]any)
	if fields == nil {
		return nil, fmt.Errorf("issue %s has no fields", key)
	}

	// Build new issue from original fields
	newFields := map[string]any{
		"project":   fields["project"],
		"issuetype": fields["issuetype"],
		"summary":   fmt.Sprintf("Clone of %s: %v", key, fields["summary"]),
	}
	if desc := fields["description"]; desc != nil {
		newFields["description"] = desc
	}
	if priority := fields["priority"]; priority != nil {
		newFields["priority"] = priority
	}
	if labels, ok := fields["labels"].([]any); ok && len(labels) > 0 {
		newFields["labels"] = labels
	}

	// Apply overrides
	for k, v := range overrides {
		newFields[k] = v
	}

	body := map[string]any{"fields": newFields}
	return c.postJSON(c.strategy.APIPath("issue"), body)
}

// AssignIssue assigns an issue to a user.
// Pass accountID for cloud, name for server. Pass empty strings for unassigned.
func (c *Client) AssignIssue(key, accountID, name, _ string) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s/assignee", key))
	return c.putNoContent(path, c.strategy.AssignBody(accountID, name))
}

// GetIssueTransitions returns available transitions for an issue.
func (c *Client) GetIssueTransitions(key string) ([]map[string]any, error) {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s/transitions", key))
	data, err := c.getJSON(path)
	if err != nil {
		return nil, err
	}
	return toSlice(data["transitions"])
}

// TransitionIssue moves an issue to a new status via transition ID.
func (c *Client) TransitionIssue(key, transitionID string) error {
	return c.TransitionIssueWithUpdates(key, transitionID, nil, nil)
}

// CommentFieldValue formats a comment body for the selected Jira API strategy.
func (c *Client) CommentFieldValue(body string) any {
	return c.strategy.CommentBody(body)
}

// AssigneeFieldValue formats an assignee field for the selected Jira API strategy.
func (c *Client) AssigneeFieldValue(accountID, name string) map[string]any {
	return c.strategy.AssigneeField(accountID, name)
}

// TransitionIssueWithUpdates moves an issue and optionally sets fields/update operations.
func (c *Client) TransitionIssueWithUpdates(key, transitionID string, fields, update map[string]any) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s/transitions", key))
	body := map[string]any{
		"transition": map[string]any{"id": transitionID},
	}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	if len(update) > 0 {
		body["update"] = update
	}
	_, err := c.postJSON(path, body)
	return err
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(key, body string) error {
	path := c.strategy.APIPath(fmt.Sprintf("issue/%s/comment", key))
	payload := map[string]any{
		"body": c.strategy.CommentBody(body),
	}
	_, err := c.postJSON(path, payload)
	return err
}

// LinkIssues creates a link between two issues.
func (c *Client) LinkIssues(inward, outward, linkType string) error {
	body := map[string]any{
		"inwardIssue":  map[string]any{"key": inward},
		"outwardIssue": map[string]any{"key": outward},
		"type":         map[string]any{"name": linkType},
	}
	_, err := c.postJSON(c.strategy.APIPath("issueLink"), body)
	return err
}

// GetIssueLinks returns the links for an issue.
func (c *Client) GetIssueLinks(key string) ([]map[string]any, error) {
	data, err := c.GetIssue(key, []string{"issuelinks"})
	if err != nil {
		return nil, err
	}
	fields, _ := data["fields"].(map[string]any)
	if fields == nil {
		return nil, nil
	}
	return toSlice(fields["issuelinks"])
}

// DeleteIssueLink deletes an issue link by ID.
func (c *Client) DeleteIssueLink(linkID string) error {
	path := c.strategy.APIPath(fmt.Sprintf("issueLink/%s", linkID))
	return c.delete(path)
}

// --- Search ---

// Search executes a JQL search query using the selected Jira REST API strategy.
// Requests key and common fields by default so results are useful.
func (c *Client) Search(jql string, startAt, maxResults int, extraFields ...string) (map[string]any, error) {
	fields := "key,summary,status,assignee,priority,issuetype,reporter,resolution,created,updated,labels,description,comment"
	if len(extraFields) > 0 {
		fields += "," + strings.Join(extraFields, ",")
	}

	searchPage := func(pageSize int, nextPageToken string) (map[string]any, error) {
		path := fmt.Sprintf("%s?jql=%s&startAt=%d&maxResults=%d&fields=%s",
			c.strategy.SearchPath(), urlEncode(jql), startAt, pageSize, fields)
		if nextPageToken != "" {
			path += "&nextPageToken=" + urlEncode(nextPageToken)
		}
		return c.getJSON(path)
	}

	data, err := searchPage(maxResults, "")
	if err != nil || !c.strategy.SearchUsesPageToken() || maxResults <= 0 {
		return data, err
	}

	issues, _ := data["issues"].([]any)
	for len(issues) < maxResults {
		nextPageToken, _ := data["nextPageToken"].(string)
		if nextPageToken == "" {
			break
		}

		page, pageErr := searchPage(maxResults-len(issues), nextPageToken)
		if pageErr != nil {
			return nil, pageErr
		}
		pageIssues, _ := page["issues"].([]any)
		issues = append(issues, pageIssues...)

		delete(data, "nextPageToken")
		for key, value := range page {
			if key != "issues" {
				data[key] = value
			}
		}
		data["issues"] = issues
	}

	return data, nil
}

// SearchWithChangelog runs a JQL search and asks Jira to expand each issue's
// changelog, so callers can inspect change history. Used by `me audit` to
// reconstruct a user's activity from issue history.
func (c *Client) SearchWithChangelog(jql string, maxResults int) (map[string]any, error) {
	path := fmt.Sprintf("%s?jql=%s&maxResults=%d&fields=%s&expand=changelog",
		c.strategy.SearchPath(), urlEncode(jql), maxResults,
		"key,summary,status,issuetype,created,updated")
	return c.getJSON(path)
}

// --- Boards ---

// ListBoards lists boards, optionally filtered by name and project.
func (c *Client) ListBoards(name string, maxResults int, projectKeyOrID string) ([]map[string]any, error) {
	path := "/rest/agile/1.0/board?"
	params := []string{fmt.Sprintf("maxResults=%d", maxResults)}
	if name != "" {
		params = append(params, "name="+name)
	}
	if projectKeyOrID != "" {
		params = append(params, "projectKeyOrId="+projectKeyOrID)
	}
	path += strings.Join(params, "&")

	data, err := c.getJSON(path)
	if err != nil {
		return nil, err
	}
	return toSlice(data["values"])
}

// GetBoard fetches a single board by ID.
func (c *Client) GetBoard(boardID int) (map[string]any, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d", boardID)
	return c.getJSON(path)
}

// FetchBoardIssues fetches issues for a board.
func (c *Client) FetchBoardIssues(boardID, startAt, maxResults int, jql string) (map[string]any, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/issue?startAt=%d&maxResults=%d", boardID, startAt, maxResults)
	if jql != "" {
		path += "&jql=" + jql
	}
	return c.getJSON(path)
}

// --- Sprints ---

// ListSprints lists sprints for a board, optionally filtered by state.
func (c *Client) ListSprints(boardID int, state string) ([]map[string]any, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint?maxResults=100", boardID)
	if state != "" {
		path += "&state=" + state
	}
	data, err := c.getJSON(path)
	if err != nil {
		return nil, err
	}
	return toSlice(data["values"])
}

// GetSprint fetches a single sprint by ID.
func (c *Client) GetSprint(sprintID int) (map[string]any, error) {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d", sprintID)
	return c.getJSON(path)
}

// GetSprintIssues fetches issues in a sprint.
func (c *Client) GetSprintIssues(sprintID int) (map[string]any, error) {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	return c.getJSON(path)
}

// StartSprint starts a sprint with the given payload.
func (c *Client) StartSprint(sprintID int, payload map[string]any) error {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d", sprintID)
	return c.putNoContent(path, payload)
}

// CloseSprint closes a sprint with the given payload.
func (c *Client) CloseSprint(sprintID int, payload map[string]any) error {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d", sprintID)
	return c.putNoContent(path, payload)
}

// MoveIssuesToSprint moves issues to a sprint.
func (c *Client) MoveIssuesToSprint(sprintID int, issueKeys []string) error {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	body := map[string]any{"issues": issueKeys}
	_, err := c.postJSON(path, body)
	return err
}

// --- Projects ---

// ListProjects returns all accessible projects.
func (c *Client) ListProjects() ([]map[string]any, error) {
	return c.getJSONArray(c.strategy.APIPath("project"))
}

// GetProject fetches a single project by key.
func (c *Client) GetProject(projectKey string) (map[string]any, error) {
	path := c.strategy.APIPath(fmt.Sprintf("project/%s", projectKey))
	return c.getJSON(path)
}

// --- Users ---

// ListUsers searches for users by query string.
func (c *Client) ListUsers(query string) ([]map[string]any, error) {
	path := c.strategy.UserSearchPath(query)
	return c.getJSONArray(path)
}

// GetUser fetches a user by account ID.
func (c *Client) GetUser(accountID string) (map[string]any, error) {
	path := c.strategy.UserPath(accountID)
	return c.getJSON(path)
}

// --- HTTP Helpers ---

func (c *Client) getJSON(path string) (map[string]any, error) {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.formatError(resp)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (c *Client) getJSONArray(path string) ([]map[string]any, error) {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.formatError(resp)
	}

	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (c *Client) postJSON(path string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", path, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.formatError(resp)
	}

	// Some POST endpoints return 204 No Content
	if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (c *Client) putNoContent(path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest("PUT", path, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.formatError(resp)
	}
	return nil
}

func (c *Client) delete(path string) error {
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.formatError(resp)
	}
	return nil
}

// Raw sends an arbitrary request to the Jira instance and returns the raw
// response. path is appended to the base URL verbatim (query string included),
// so any REST resource is reachable — including ones this CLI has no
// predefined command for. Extra headers override the defaults.
func (c *Client) Raw(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	return c.do(method, path, body, headers)
}

// APIPath prefixes a version-relative resource (e.g. "issue/PROJ-1") with the
// platform REST base path for this client's flavor: /rest/api/3 on Cloud,
// /rest/api/2 on Server/Data Center.
func (c *Client) APIPath(resource string) string {
	return c.strategy.APIPath(resource)
}

// IsCloud reports whether the client talks to Jira Cloud (REST API v3)
// rather than Jira Server/Data Center (REST API v2).
func (c *Client) IsCloud() bool {
	_, ok := c.strategy.(cloudStrategy)
	return ok
}

func (c *Client) doRequest(method, path string, body []byte) (*http.Response, error) {
	return c.do(method, path, body, nil)
}

func (c *Client) do(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set auth
	switch c.authType {
	case "basic":
		req.SetBasicAuth(c.email, c.token)
	case "pat", "bearer":
		req.Header.Set("Authorization", "Bearer "+c.token)
	default:
		// Default to basic for cloud (*.atlassian.net), bearer otherwise
		if strings.Contains(c.BaseURL, ".atlassian.net") {
			req.SetBasicAuth(c.email, c.token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	c.writeDebugResponse(req, resp)
	return resp, nil
}

func (c *Client) writeDebugResponse(req *http.Request, resp *http.Response) {
	if c.debug == nil {
		return
	}

	originalBody := resp.Body
	body, readErr := io.ReadAll(originalBody)
	if readErr == nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		resp.Body = &replayReadCloser{
			Reader: io.MultiReader(bytes.NewReader(body), originalBody),
			Closer: originalBody,
		}
	}

	headers := resp.Header.Clone()
	for name := range headers {
		if strings.EqualFold(name, "Set-Cookie") {
			headers[name] = []string{"[REDACTED]"}
		}
	}

	var debug bytes.Buffer
	fmt.Fprintln(&debug, "--- jira debug response ---")
	fmt.Fprintf(&debug, "%s %s\n", req.Method, req.URL.String())
	fmt.Fprintf(&debug, "%s %s\n", resp.Proto, resp.Status)
	_ = headers.Write(&debug)
	fmt.Fprintln(&debug)
	_, _ = debug.Write(body)
	if len(body) == 0 || body[len(body)-1] != '\n' {
		fmt.Fprintln(&debug)
	}
	if readErr != nil {
		fmt.Fprintf(&debug, "[debug: reading response body failed: %v]\n", readErr)
	}
	fmt.Fprintln(&debug, "--- end jira debug response ---")
	_, _ = c.debug.Write(debug.Bytes())
}

type replayReadCloser struct {
	io.Reader
	io.Closer
}

func (c *Client) formatError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse Jira error response
	var jiraErr struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if json.Unmarshal(body, &jiraErr) == nil {
		var parts []string
		parts = append(parts, jiraErr.ErrorMessages...)
		for k, v := range jiraErr.Errors {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v))
		}
		if len(parts) > 0 {
			return fmt.Errorf("jira: %s (%d)", strings.Join(parts, "; "), resp.StatusCode)
		}
	}

	return fmt.Errorf("jira: unexpected status %d: %s", resp.StatusCode, string(body))
}

// toSlice converts an any value (expected []any) to []map[string]any.
func toSlice(v any) ([]map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", v)
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// urlEncode percent-encodes a string for use in a URL query parameter.
func urlEncode(s string) string {
	return url.QueryEscape(s)
}
