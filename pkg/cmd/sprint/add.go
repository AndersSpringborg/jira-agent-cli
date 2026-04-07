package sprint

import (
	"fmt"
	"strconv"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newAddCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <sprint-id> <issue-key>...",
		Short: "Add issues to a sprint",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid sprint ID: %s", args[0])
			}

			issueKeys := make([]string, len(args)-1)
			for i, key := range args[1:] {
				issueKeys[i] = strings.ToUpper(key)
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			if err := client.MoveIssuesToSprint(sprintID, issueKeys); err != nil {
				return err
			}

			fmt.Printf("Added %d issue(s) to sprint %d\n", len(issueKeys), sprintID)
			return nil
		},
	}

	return cmd
}
