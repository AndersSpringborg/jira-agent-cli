package sprint

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newStartCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name      string
		startDate string
		endDate   string
		goal      string
	)

	cmd := &cobra.Command{
		Use:   "start <sprint-id>",
		Short: "Start a sprint",
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
				"state": "active",
			}
			if name != "" {
				payload["name"] = name
			}
			if startDate != "" {
				payload["startDate"] = startDate
			}
			if endDate != "" {
				payload["endDate"] = endDate
			}
			if goal != "" {
				payload["goal"] = goal
			}

			if err := client.StartSprint(sprintID, payload); err != nil {
				return err
			}

			fmt.Printf("Started sprint: %d\n", sprintID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Sprint name")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (ISO 8601)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (ISO 8601)")
	cmd.Flags().StringVar(&goal, "goal", "", "Sprint goal")

	return cmd
}
