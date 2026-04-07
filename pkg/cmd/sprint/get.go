package sprint

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newGetCmd(f *cmdutil.Factory) *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "get <sprint-id>",
		Short: "Get sprint details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID: %s", args[0])
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.GetSprint(sprintID)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(data)
			}

			return driver.Item("Sprint", data)
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON")

	return cmd
}
