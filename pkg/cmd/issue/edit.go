package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		summary     string
		description string
		priority    string
		labels      []string
		components  []string
		fixVersions []string
		noInput     bool
	)

	cmd := &cobra.Command{
		Use:     "edit <issue-key>",
		Aliases: []string{"update"},
		Short:   "Edit an issue",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			if summary == "" && description == "" && priority == "" && len(labels) == 0 && len(components) == 0 && len(fixVersions) == 0 {
				return fmt.Errorf("at least one field to update is required")
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			fields := map[string]any{}
			if summary != "" {
				fields["summary"] = summary
			}
			if description != "" {
				fields["description"] = description
			}
			if priority != "" {
				fields["priority"] = map[string]any{"name": priority}
			}

			if len(labels) > 0 {
				var add []map[string]any
				var remove []map[string]any
				for _, l := range labels {
					if strings.HasPrefix(l, "-") {
						remove = append(remove, map[string]any{"remove": l[1:]})
					} else {
						add = append(add, map[string]any{"add": l})
					}
				}
				update := map[string]any{}
				if len(add) > 0 || len(remove) > 0 {
					ops := make([]map[string]any, 0, len(add)+len(remove))
					ops = append(ops, add...)
					ops = append(ops, remove...)
					update["labels"] = ops
					fields["__update_labels"] = true
				}
			}

			if len(components) > 0 {
				var add []map[string]any
				var remove []map[string]any
				for _, c := range components {
					if strings.HasPrefix(c, "-") {
						remove = append(remove, map[string]any{"remove": map[string]any{"name": c[1:]}})
					} else {
						add = append(add, map[string]any{"add": map[string]any{"name": c}})
					}
				}
				ops := make([]map[string]any, 0, len(add)+len(remove))
				ops = append(ops, add...)
				ops = append(ops, remove...)
				fields["__update_components"] = ops
			}

			if len(fixVersions) > 0 {
				var add []map[string]any
				var remove []map[string]any
				for _, v := range fixVersions {
					if strings.HasPrefix(v, "-") {
						remove = append(remove, map[string]any{"remove": map[string]any{"name": v[1:]}})
					} else {
						add = append(add, map[string]any{"add": map[string]any{"name": v}})
					}
				}
				ops := make([]map[string]any, 0, len(add)+len(remove))
				ops = append(ops, add...)
				ops = append(ops, remove...)
				fields["__update_fixVersions"] = ops
			}

			if err := client.UpdateIssue(issueKey, fields); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Updated issue: %s", issueKey)
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Issue summary")
	cmd.Flags().StringVarP(&description, "body", "b", "", "Issue description")
	cmd.Flags().StringVarP(&priority, "priority", "y", "", "Issue priority")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Label (prefix - to remove, repeatable)")
	cmd.Flags().StringSliceVarP(&components, "component", "C", nil, "Component (prefix - to remove, repeatable)")
	cmd.Flags().StringSliceVar(&fixVersions, "fix-version", nil, "Fix version (prefix - to remove, repeatable)")
	cmd.Flags().BoolVar(&noInput, "no-input", false, "Disable interactive prompt")
	_ = noInput

	return cmd
}
