// Package api implements `jira api`: raw passthrough access to any Jira REST
// endpoint, plus `--list` to discover what the API offers. It is the escape
// hatch for everything the predefined commands don't cover.
package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"AndersSpringborg/jira-cli/internal/apispec"
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		method  string
		data    string
		headers []string
		list    bool
	)

	cmd := &cobra.Command{
		Use:   "api [<path> | --list [filter...]]",
		Short: "Raw access to any Jira REST API endpoint",
		Long: `Send a raw request to any Jira REST API endpoint using the
authentication from the active profile, and print the response body verbatim.

Paths starting with "/" are used as-is, so every REST resource is reachable
(platform, agile, plugins, experimental). Paths without a leading "/" are
treated as platform resources and prefixed with the API version matching the
profile: /rest/api/3 on Jira Cloud, /rest/api/2 on Jira Server/Data Center.

Use --list to browse the available endpoints for your instance flavor
(no authentication required).`,
		Example: `  # GET is the default; full paths pass through verbatim
  jira api /rest/agile/1.0/board/42/backlog
  jira api "/rest/api/3/issue/PROJ-1?expand=changelog,renderedFields"

  # shorthand: version prefix deduced from the profile (v3 cloud, v2 server)
  jira api issue/PROJ-1

  # writes: -d implies POST; use @file or - (stdin) for the body
  jira api issue -d '{"fields":{...}}'
  jira api -X PUT issue/PROJ-1 -d @payload.json
  echo '{"issues":["PROJ-1"]}' | jira api /rest/agile/1.0/sprint/7/issue -d -

  # discover endpoints (filter terms match method, path, and description)
  jira api --list sprint
  jira api --list "POST issue"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return runList(f, cmd, strings.Join(args, " "))
			}
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one <path> argument (or --list); see `jira api --help`")
			}
			return runRequest(f, cmd, args[0], method, data, headers)
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "", "HTTP method (default GET, or POST when a body is given)")
	cmd.Flags().StringVarP(&data, "data", "d", "", "request body: a raw string, @file, or - for stdin")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "extra request header in 'Key: Value' form (repeatable)")
	cmd.Flags().BoolVar(&list, "list", false, "list known API endpoints for the profile's flavor instead of sending a request")

	return cmd
}

func runRequest(f *cmdutil.Factory, cmd *cobra.Command, path, method, data string, headers []string) error {
	client, err := f.LoadClient()
	if err != nil {
		return err
	}

	body, err := readBody(cmd, data)
	if err != nil {
		return err
	}

	if method == "" {
		method = http.MethodGet
		if body != nil {
			method = http.MethodPost
		}
	}
	method = strings.ToUpper(method)

	if !strings.HasPrefix(path, "/") {
		path = client.APIPath(path)
	}

	hdrs, err := parseHeaders(headers)
	if err != nil {
		return err
	}

	resp, err := client.Raw(method, path, body, hdrs)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	out := cmd.OutOrStdout()
	if len(respBody) > 0 {
		if _, err := out.Write(respBody); err != nil {
			return err
		}
		if respBody[len(respBody)-1] != '\n' {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
	}

	if resp.StatusCode >= 400 {
		// The Jira error body was already printed; the exit status signals failure.
		cmd.SilenceUsage = true
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func readBody(cmd *cobra.Command, data string) ([]byte, error) {
	switch {
	case !cmd.Flags().Changed("data"):
		return nil, nil
	case data == "-":
		body, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read body from stdin: %w", err)
		}
		return body, nil
	case strings.HasPrefix(data, "@"):
		body, err := os.ReadFile(strings.TrimPrefix(data, "@"))
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		return body, nil
	default:
		return []byte(data), nil
	}
}

func parseHeaders(headers []string) (map[string]string, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	hdrs := make(map[string]string, len(headers))
	for _, h := range headers {
		key, value, ok := strings.Cut(h, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid header %q: expected 'Key: Value'", h)
		}
		hdrs[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return hdrs, nil
}

func runList(f *cmdutil.Factory, cmd *cobra.Command, filter string) error {
	endpoints, err := apispec.List(profileFlavor(f), filter)
	if err != nil {
		return err
	}

	rows := make([]map[string]any, len(endpoints))
	for i, e := range endpoints {
		rows[i] = map[string]any{
			"method":  e.Method,
			"path":    e.Path,
			"summary": e.Summary,
		}
	}

	driver := f.DisplayDriverTo(cmd, cmd.OutOrStdout())
	return driver.List("API Endpoints", []string{"method", "path", "summary"}, rows)
}

// profileFlavor deduces the API flavor from the active profile without
// requiring a token, mirroring how the client selects its strategy:
// PAT/bearer auth means Server/Data Center (v2), anything else is Cloud (v3).
func profileFlavor(f *cmdutil.Factory) apispec.Flavor {
	authType := os.Getenv("JIRA_AUTH_TYPE")
	baseURL := os.Getenv("JIRA_BASE_URL")

	if profile, err := f.LoadProfile(); err == nil {
		if authType == "" {
			authType = profile.AuthType
		}
		if baseURL == "" {
			baseURL = profile.BaseURL
		}
	}
	if authType == "" && baseURL != "" {
		authType = config.DetectAuthType(baseURL)
	}

	if authType == "pat" || authType == "bearer" {
		return apispec.FlavorServer
	}
	return apispec.FlavorCloud
}
