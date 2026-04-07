package issue

import (
	"fmt"
	"os/exec"
	"runtime"
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

			var openCmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				openCmd = exec.Command("open", issueURL)
			case "linux":
				openCmd = exec.Command("xdg-open", issueURL)
			case "windows":
				openCmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", issueURL)
			default:
				fmt.Println(issueURL)
				return nil
			}

			if err := openCmd.Start(); err != nil {
				fmt.Println(issueURL)
				return nil
			}
			fmt.Printf("Opening %s in browser...\n", issueKey)
			return nil
		},
	}

	return cmd
}
