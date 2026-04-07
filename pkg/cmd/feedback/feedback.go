package feedback

import (
	"fmt"
	"net/url"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

const repoURL = "https://github.com/AndersSpringborg/jira-agent-cli"

// NewCmd creates the feedback command.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feedback",
		Short: "Open a GitHub issue to report a bug or request a feature",
		Long: `Opens a pre-filled GitHub issue form in your default browser.

Examples:
  jira feedback --title "ADF rendering broken"
  jira feedback --title "Feature: sprint burndown" --body "Would be nice to see burndown charts"
  jira feedback --title "Bug report" --no-browser   # just print the URL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			driver := f.DisplayDriver(cmd)

			title, _ := cmd.Flags().GetString("title")
			body, _ := cmd.Flags().GetString("body")
			noBrowser, _ := cmd.Flags().GetBool("no-browser")

			issueURL := BuildIssueURL(title, body)

			if noBrowser {
				return driver.Raw(map[string]any{"url": issueURL})
			}

			if err := cmdutil.OpenBrowser(issueURL); err != nil {
				// Browser failed — output the URL so the user/agent can use it.
				return driver.Raw(map[string]any{"url": issueURL})
			}

			return driver.Message("Opening feedback form: %s", issueURL)
		},
	}

	cmd.Flags().String("title", "", "Issue title (required)")
	cmd.Flags().String("body", "", "Issue body / description")
	cmd.Flags().Bool("no-browser", false, "Print the URL instead of opening a browser")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

// BuildIssueURL constructs a GitHub new-issue URL with pre-filled title and body.
func BuildIssueURL(title, body string) string {
	u := fmt.Sprintf("%s/issues/new", repoURL)

	params := url.Values{}
	params.Set("title", title)
	if body != "" {
		params.Set("body", body)
	}

	return u + "?" + params.Encode()
}
