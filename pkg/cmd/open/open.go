package open

import (
	"fmt"
	"os/exec"
	"runtime"
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

			var openCmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				openCmd = exec.Command("open", targetURL)
			case "linux":
				openCmd = exec.Command("xdg-open", targetURL)
			case "windows":
				openCmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
			default:
				fmt.Println(targetURL)
				return nil
			}

			if err := openCmd.Start(); err != nil {
				fmt.Println(targetURL)
				return nil
			}
			fmt.Printf("Opening %s\n", targetURL)
			return nil
		},
	}

	return cmd
}
