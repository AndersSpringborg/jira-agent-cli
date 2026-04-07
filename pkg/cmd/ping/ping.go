package ping

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	var format string

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

			if format == "json" {
				return output.JSON(result)
			}

			output.Table([]map[string]any{result}, []string{"ok", "accountId"}, "Ping")
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")

	return cmd
}
