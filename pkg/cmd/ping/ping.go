package ping

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity to Jira",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetMyself()
			if err != nil {
				return err
			}

			result := map[string]any{
				"ok":        true,
				"accountId": data["accountId"],
			}

			driver := f.DisplayDriver(cmd)
			return driver.Item("Ping", result)
		},
	}

	return cmd
}
