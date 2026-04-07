package project

import (
	"strings"

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
		Use:   "get <project-key>",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetProject(projectKey)
			if err != nil {
				return err
			}

			if format == "json" || raw {
				return output.JSON(data)
			}

			cols := []string{"key", "name", "projectTypeKey", "lead"}
			output.Table([]map[string]any{data}, cols, "Project")
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
