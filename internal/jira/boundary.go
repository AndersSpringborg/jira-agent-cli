package jira

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"AndersSpringborg/jira-cli/internal/config"
)

// ErrOutsideBoundary is returned when a request would see or change issues
// outside the user's protected boundary.
var ErrOutsideBoundary = errors.New("outside the jira boundary")

const boundaryHint = "The user restricted this CLI to a boundary (see `jira boundary show`). " +
	"AI agents: do not try to work around it; if you need access, ask the user."

var issueKeyPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*-[0-9]+|[0-9]+)$`)

// SetBoundary confines every subsequent request to b. An inactive boundary
// leaves the client unrestricted.
func (c *Client) SetBoundary(b *config.Boundary) {
	if b.Active() {
		c.boundary = b
	}
}

func outside(format string, args ...any) error {
	return fmt.Errorf("%s is %w. %s", fmt.Sprintf(format, args...), ErrOutsideBoundary, boundaryHint)
}

// scopeJQL rewrites jql to `(<boundary>) AND (<where>) ORDER BY ...`.
func (c *Client) scopeJQL(jql string) (string, error) {
	if c.boundary == nil {
		return jql, nil
	}
	where, order, err := splitOrderBy(jql)
	if err != nil {
		return "", fmt.Errorf("JQL cannot be confined to the boundary (%v): %w. %s", err, ErrOutsideBoundary, boundaryHint)
	}
	scoped := c.boundary.JQL()
	if where = strings.TrimSpace(where); where != "" {
		scoped = "(" + scoped + ") AND (" + where + ")"
	}
	if order != "" {
		scoped += " " + order
	}
	return scoped, nil
}

// splitOrderBy splits jql at its top-level ORDER BY, rejecting input whose
// quotes or parentheses are unbalanced and could escape the wrapping.
func splitOrderBy(jql string) (where, order string, err error) {
	depth := 0
	var quote rune
	orderAt := -1
	runes := []rune(jql)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote != 0:
			switch r {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '(':
			depth++
		case r == ')':
			depth--
			if depth < 0 {
				return "", "", errors.New("unbalanced parentheses")
			}
		case depth == 0 && orderAt < 0 && startsOrderBy(runes, i):
			orderAt = i
		}
	}
	if quote != 0 {
		return "", "", errors.New("unterminated quote")
	}
	if depth != 0 {
		return "", "", errors.New("unbalanced parentheses")
	}
	if orderAt < 0 {
		return jql, "", nil
	}
	return string(runes[:orderAt]), strings.TrimSpace(string(runes[orderAt:])), nil
}

func startsOrderBy(runes []rune, i int) bool {
	if i > 0 && !unicode.IsSpace(runes[i-1]) && runes[i-1] != ')' {
		return false
	}
	rest := strings.ToLower(string(runes[i:]))
	if !strings.HasPrefix(rest, "order") {
		return false
	}
	rest = rest[len("order"):]
	trimmed := strings.TrimLeftFunc(rest, unicode.IsSpace)
	return len(trimmed) < len(rest) && strings.HasPrefix(trimmed, "by") &&
		(len(trimmed) == 2 || unicode.IsSpace(rune(trimmed[2])))
}

// confine drops issues outside the boundary from a search-style response
// and redacts embedded references to outside issues in the rest.
func (c *Client) confine(data map[string]any) map[string]any {
	if c.boundary == nil || data == nil {
		return data
	}
	raw, _ := data["issues"].([]any)
	kept := make([]any, 0, len(raw))
	issues := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if issue, ok := item.(map[string]any); ok && c.boundary.Allows(issue) {
			kept = append(kept, issue)
			issues = append(issues, issue)
		}
	}
	data["issues"] = kept
	c.redact(issues...)
	return data
}

// allowedKeys returns the subset of keys (or IDs) inside the boundary.
func (c *Client) allowedKeys(keys []string) (map[string]bool, error) {
	for _, key := range keys {
		if !issueKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid issue key %q", key)
		}
	}
	jql, err := c.scopeJQL("issuekey in (" + strings.ToUpper(strings.Join(keys, ", ")) + ")")
	if err != nil {
		return nil, err
	}
	data, err := c.search(jql, 0, len(keys), "key,project,labels")
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	issues, _ := data["issues"].([]any)
	for _, item := range issues {
		issue, _ := item.(map[string]any)
		if issue == nil || !c.boundary.Allows(issue) {
			continue
		}
		if key, ok := issue["key"].(string); ok {
			allowed[strings.ToUpper(key)] = true
		}
		if id, ok := issue["id"].(string); ok {
			allowed[id] = true
		}
	}
	return allowed, nil
}

// requireIssues fails unless every key is inside the boundary.
func (c *Client) requireIssues(keys ...string) error {
	if c.boundary == nil {
		return nil
	}
	allowed, err := c.allowedKeys(dedupe(keys))
	if err != nil {
		return err
	}
	for _, key := range keys {
		if !allowed[strings.ToUpper(key)] {
			return outside("issue %s", key)
		}
	}
	return nil
}

func dedupe(keys []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if upper := strings.ToUpper(key); !seen[upper] {
			seen[upper] = true
			out = append(out, key)
		}
	}
	return out
}

// redact removes linked issues, subtasks, and parents outside the boundary,
// so their summaries do not leak through issues inside it. If the check
// fails, every embedded reference is removed.
func (c *Client) redact(issues ...map[string]any) {
	if c.boundary == nil {
		return
	}
	var keys []string
	for _, issue := range issues {
		keys = append(keys, embeddedKeys(issue)...)
	}
	if len(keys) == 0 {
		return
	}
	allowed, err := c.allowedKeys(dedupe(keys))
	if err != nil {
		allowed = map[string]bool{}
	}
	isAllowed := func(ref any) bool {
		m, _ := ref.(map[string]any)
		key, _ := m["key"].(string)
		return allowed[strings.ToUpper(key)]
	}
	for _, issue := range issues {
		fields, _ := issue["fields"].(map[string]any)
		if fields == nil {
			continue
		}
		if links, ok := fields["issuelinks"].([]any); ok {
			kept := []any{}
			for _, link := range links {
				l, _ := link.(map[string]any)
				ref := l["inwardIssue"]
				if ref == nil {
					ref = l["outwardIssue"]
				}
				if isAllowed(ref) {
					kept = append(kept, link)
				}
			}
			fields["issuelinks"] = kept
		}
		if subtasks, ok := fields["subtasks"].([]any); ok {
			kept := []any{}
			for _, sub := range subtasks {
				if isAllowed(sub) {
					kept = append(kept, sub)
				}
			}
			fields["subtasks"] = kept
		}
		if parent, ok := fields["parent"]; ok && !isAllowed(parent) {
			delete(fields, "parent")
		}
	}
}

func embeddedKeys(issue map[string]any) []string {
	fields, _ := issue["fields"].(map[string]any)
	var keys []string
	add := func(ref any) {
		if m, ok := ref.(map[string]any); ok {
			if key, ok := m["key"].(string); ok && issueKeyPattern.MatchString(key) {
				keys = append(keys, key)
			}
		}
	}
	links, _ := fields["issuelinks"].([]any)
	for _, link := range links {
		l, _ := link.(map[string]any)
		add(l["inwardIssue"])
		add(l["outwardIssue"])
	}
	subtasks, _ := fields["subtasks"].([]any)
	for _, sub := range subtasks {
		add(sub)
	}
	add(fields["parent"])
	return keys
}

// checkNewIssue ensures a new issue's project is inside the boundary and adds
// the boundary's required labels.
func (c *Client) checkNewIssue(fields map[string]any) error {
	if c.boundary == nil {
		return nil
	}
	project := projectKey(fields["project"])
	if !c.boundary.AllowsProject(project) {
		return outside("project %q", project)
	}
	if parent := projectKey(fields["parent"]); parent != "" {
		if err := c.requireIssues(parent); err != nil {
			return err
		}
	}
	labels := labelStrings(fields["labels"])
	for _, required := range c.boundary.Labels {
		if !containsString(labels, required) {
			labels = append(labels, required)
		}
	}
	if len(labels) > 0 {
		fields["labels"] = labels
	}
	return nil
}

// checkEdit requires key inside the boundary and its labels to stay there.
func (c *Client) checkEdit(key string, fields, update map[string]any) error {
	if err := c.checkLabelChanges(fields, update); err != nil {
		return err
	}
	return c.requireIssues(key)
}

// checkLabelChanges rejects edits that would remove a boundary label and so
// move an issue out of the boundary.
func (c *Client) checkLabelChanges(fields, update map[string]any) error {
	if c.boundary == nil || len(c.boundary.Labels) == 0 {
		return nil
	}
	if labels, ok := fields["labels"]; ok {
		if err := c.requireBoundaryLabels(labelStrings(labels)); err != nil {
			return err
		}
	}
	ops, _ := update["labels"].([]any)
	for _, op := range ops {
		m, _ := op.(map[string]any)
		if removed, ok := m["remove"].(string); ok && containsString(c.boundary.Labels, removed) {
			return outside("removing boundary label %q", removed)
		}
		if set, ok := m["set"]; ok {
			if err := c.requireBoundaryLabels(labelStrings(set)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) requireBoundaryLabels(labels []string) error {
	for _, required := range c.boundary.Labels {
		if !containsString(labels, required) {
			return outside("dropping boundary label %q", required)
		}
	}
	return nil
}

func projectKey(v any) string {
	switch p := v.(type) {
	case map[string]any:
		key, _ := p["key"].(string)
		return key
	case map[string]string:
		return p["key"]
	case string:
		return p
	}
	return ""
}

func labelStrings(v any) []string {
	switch labels := v.(type) {
	case []string:
		return append([]string(nil), labels...)
	case []any:
		out := make([]string, 0, len(labels))
		for _, l := range labels {
			if s, ok := l.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
