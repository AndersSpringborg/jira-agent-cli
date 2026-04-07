package sprint

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newCloseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <sprint-id>",
		Short: "Close a sprint",
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

			payload := map[string]any{
				"state": "closed",
			}

			if err := client.CloseSprint(sprintID, payload); err != nil {
				return err
			}

			fmt.Printf("Closed sprint: %d\n", sprintID)
			return nil
		},
	}

	return cmd
}
