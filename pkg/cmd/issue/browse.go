package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newBrowseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse <issue-key>",
		Short: "Open an issue in the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			issueURL := fmt.Sprintf("%s/browse/%s", client.BaseURL, issueKey)
			driver := f.DisplayDriver(cmd)

			if err := cmdutil.OpenBrowser(issueURL); err != nil {
				return driver.Message("%s", issueURL)
			}
			return driver.Message("Opening %s in browser...", issueKey)
		},
	}

	return cmd
}
