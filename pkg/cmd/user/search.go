package user

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newSearchCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		columns string
		raw     bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			users, err := client.ListUsers(query)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(users)
			}

			cols := output.NormalizeFields(columns, []string{"accountId", "displayName", "emailAddress"})
			return driver.List("Users", cols, users)
		},
	}

	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
