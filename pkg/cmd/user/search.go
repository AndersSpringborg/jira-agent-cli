package user

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newSearchCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		plain     bool
		noHeaders bool
		columns   string
		csvOutput bool
		raw       bool
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

			if raw {
				return output.JSON(users)
			}

			cols := output.NormalizeFields(columns, []string{"accountId", "displayName", "emailAddress"})
			output.TableWithOptions(users, cols, "Users", output.TableOptions{
				Plain:     plain,
				NoHeaders: noHeaders,
				CSV:       csvOutput,
			})
			return nil
		},
	}

	cmd.Flags().BoolVar(&plain, "plain", false, "Plain output (tab-separated)")
	cmd.Flags().BoolVar(&noHeaders, "no-headers", false, "Don't print column headers")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&csvOutput, "csv", false, "Output in CSV format")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
