package user

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newGetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format string
		raw    bool
	)

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

			if format == "json" || raw {
				return output.JSON(data)
			}

			cols := []string{"accountId", "displayName", "emailAddress", "active"}
			output.Table([]map[string]any{data}, cols, "User")
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
