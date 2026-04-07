package auth

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newWhoamiCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format  string
		noColor bool
		raw     bool
	)

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetMyself()
			if err != nil {
				return err
			}

			if format == "json" || raw {
				return output.JSON(data)
			}

			row := map[string]any{
				"displayName":  data["displayName"],
				"emailAddress": data["emailAddress"],
				"accountId":    data["accountId"],
			}
			_ = noColor
			output.Table([]map[string]any{row}, []string{"displayName", "emailAddress", "accountId"}, "Current User")
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json, ndjson)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable color")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
