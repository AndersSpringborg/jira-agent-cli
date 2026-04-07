package user

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newGetCmd(f *cmdutil.Factory) *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "get <account-id>",
		Short: "Get user details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := args[0]

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetUser(accountID)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(data)
			}

			return driver.Item("User", data)
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
