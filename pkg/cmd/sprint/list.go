package sprint

import (
	"fmt"
	"strconv"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		state   string
		current bool
		prev    bool
		next    bool
		columns string
		raw     bool
	)

	cmd := &cobra.Command{
		Use:   "list [board-id]",
		Short: "List sprints for a board",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var boardID int

			if len(args) > 0 {
				id, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid board ID: %s", args[0])
				}
				boardID = id
			} else {
				profile, err := f.LoadProfile()
				if err != nil {
					return err
				}
				if profile.Context != nil && profile.Context.BoardID != 0 {
					boardID = profile.Context.BoardID
				}
			}

			if boardID == 0 {
				return fmt.Errorf("board ID is required (pass as argument or set via `jira context set --board-id`)")
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			effectiveState := state
			if current {
				effectiveState = "active"
			} else if prev || next {
				effectiveState = ""
			}

			sprints, err := client.ListSprints(boardID, effectiveState)
			if err != nil {
				return err
			}

			switch {
			case current:
				var filtered []map[string]any
				for _, s := range sprints {
					if st, _ := s["state"].(string); st == "active" {
						filtered = append(filtered, s)
					}
				}
				sprints = filtered
			case prev:
				var closed []map[string]any
				for _, s := range sprints {
					if st, _ := s["state"].(string); st == "closed" {
						closed = append(closed, s)
					}
				}
				if len(closed) > 0 {
					sprints = closed[len(closed)-1:]
				} else {
					sprints = nil
				}
			case next:
				var future []map[string]any
				for _, s := range sprints {
					if st, _ := s["state"].(string); st == "future" {
						future = append(future, s)
					}
				}
				if len(future) > 0 {
					sprints = future[:1]
				} else {
					sprints = nil
				}
			}

			driver := f.DisplayDriver(cmd)

			if raw {
				return driver.Raw(sprints)
			}

			cols := output.NormalizeFields(columns, []string{"id", "name", "state", "startDate", "endDate"})
			return driver.List("Sprints", cols, sprints)
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state (active, closed, future)")
	cmd.Flags().BoolVar(&current, "current", false, "Show current active sprint")
	cmd.Flags().BoolVar(&prev, "prev", false, "Show previous sprint")
	cmd.Flags().BoolVar(&next, "next", false, "Show next planned sprint")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns")
	cmd.Flags().BoolVar(&raw, "raw", false, "Raw JSON output")

	return cmd
}
