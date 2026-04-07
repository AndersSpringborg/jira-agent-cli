package me

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"

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

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(data)
			}

			return driver.Message("%s", data["displayName"])
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
