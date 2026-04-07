package project

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		plain     bool
		noHeaders bool
		columns   string
		csvOutput bool
		raw       bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			projects, err := client.ListProjects()
			if err != nil {
				return err
			}

			if raw {
				return output.JSON(projects)
			}

			cols := output.NormalizeFields(columns, []string{"key", "name", "projectTypeKey"})
			output.TableWithOptions(projects, cols, "Projects", output.TableOptions{
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
