package board

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name           string
		projectKeyOrID string
		maxResults     int
		plain          bool
		noHeaders      bool
		columns        string
		csvOutput      bool
		raw            bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectKeyOrID == "" {
				profile, err := f.LoadProfile()
				if err != nil {
					return err
				}
				if profile.Context != nil && profile.Context.Project != "" {
					projectKeyOrID = profile.Context.Project
				}
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			boards, err := client.ListBoards(name, maxResults, projectKeyOrID)
			if err != nil {
				return err
			}

			if raw {
				return output.JSON(boards)
			}

			cols := output.NormalizeFields(columns, []string{"id", "name", "type"})
			output.TableWithOptions(boards, cols, "Boards", output.TableOptions{
				Plain:     plain,
				NoHeaders: noHeaders,
				CSV:       csvOutput,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Filter by board name")
	cmd.Flags().StringVarP(&projectKeyOrID, "project", "p", "", "Filter by project key or ID")
	cmd.Flags().IntVar(&maxResults, "max", 50, "Max results")
	cmd.Flags().BoolVar(&plain, "plain", false, "Plain output (tab-separated)")
	cmd.Flags().BoolVar(&noHeaders, "no-headers", false, "Don't print column headers")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&csvOutput, "csv", false, "Output in CSV format")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
