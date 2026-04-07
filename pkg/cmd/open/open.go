package open

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [issue-key]",
		Short: "Open project or issue in the browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			var targetURL string
			if len(args) == 0 {
				profile, err := f.LoadProfile()
				if err != nil {
					return err
				}
				if profile.Context != nil && profile.Context.Project != "" {
					targetURL = fmt.Sprintf("%s/browse/%s", client.BaseURL, profile.Context.Project)
				} else {
					targetURL = client.BaseURL
				}
			} else {
				issueKey := strings.ToUpper(args[0])
				targetURL = fmt.Sprintf("%s/browse/%s", client.BaseURL, issueKey)
			}

			driver := f.DisplayDriver(cmd)

			if err := cmdutil.OpenBrowser(targetURL); err != nil {
				return driver.Message("%s", targetURL)
			}
			return driver.Message("Opening %s", targetURL)
		},
	}

	return cmd
}
