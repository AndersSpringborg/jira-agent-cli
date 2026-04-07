package me

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutil.Factory) *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user's display name",
		Long:  "Print the display name of the authenticated user. Useful in scripts, e.g. -a$(jira me)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetMyself()
			if err != nil {
				return err
			}

			if raw {
				return output.JSON(data)
			}

			fmt.Println(data["displayName"])
			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
