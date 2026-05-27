package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		summary      string
		description  string
		priority     string
		labels       []string
		components   []string
		fixVersions  []string
		customFields []string
		noInput      bool
	)

	cmd := &cobra.Command{
		Use:     "edit <issue-key>",
		Aliases: []string{"update"},
		Short:   "Edit an issue",
		Long: `Edit an issue's fields. Standard fields use dedicated flags.
Custom fields use --field with the raw field ID:

  jira issue edit PROJ-123 --field customfield_10001=5
  jira issue edit PROJ-123 --field customfield_10002="Option A"

Array fields (labels, components, fix-versions) support add/remove
with a - prefix: --label bugfix --label -wontfix`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			hasStandardFields := summary != "" || description != "" || priority != "" ||
				len(labels) > 0 || len(components) > 0 || len(fixVersions) > 0
			if !hasStandardFields && len(customFields) == 0 {
				return fmt.Errorf("at least one field to update is required")
			}

			// Parse custom fields: --field key=value
			parsedCustom, err := parseCustomFields(customFields)
			if err != nil {
				return err
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			// "fields" for simple set operations; "update" for array add/remove.
			fields := map[string]any{}
			update := map[string]any{}

			if summary != "" {
				fields["summary"] = summary
			}
			if description != "" {
				fields["description"] = description
			}
			if priority != "" {
				fields["priority"] = map[string]any{"name": priority}
			}

			// Custom fields go into the fields section as simple set.
			for k, v := range parsedCustom {
				fields[k] = v
			}

			// Labels use the update section for add/remove operations.
			if len(labels) > 0 {
				ops := buildStringOps(labels)
				update["labels"] = ops
			}

			// Components use the update section with {name: ...} objects.
			if len(components) > 0 {
				ops := buildNamedOps(components)
				update["components"] = ops
			}

			// Fix versions use the update section with {name: ...} objects.
			if len(fixVersions) > 0 {
				ops := buildNamedOps(fixVersions)
				update["fixVersions"] = ops
			}

			if err := client.EditIssue(issueKey, fields, update); err != nil {
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
	cmd.Flags().StringArrayVarP(&customFields, "field", "F", nil, `Custom field as key=value (e.g. customfield_10001=5), repeatable`)
	cmd.Flags().BoolVar(&noInput, "no-input", false, "Disable interactive prompt")
	_ = noInput

	return cmd
}

// parseCustomFields splits "key=value" entries from --field flags.
func parseCustomFields(raw []string) (map[string]string, error) {
	result := make(map[string]string, len(raw))
	for _, entry := range raw {
		k, v, ok := strings.Cut(entry, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --field format %q: expected key=value", entry)
		}
		result[k] = v
	}
	return result, nil
}

// buildStringOps builds update operations for simple string arrays (labels).
// Entries prefixed with - become remove operations; others become add.
func buildStringOps(values []string) []map[string]any {
	ops := make([]map[string]any, 0, len(values))
	for _, v := range values {
		if strings.HasPrefix(v, "-") {
			ops = append(ops, map[string]any{"remove": v[1:]})
		} else {
			ops = append(ops, map[string]any{"add": v})
		}
	}
	return ops
}

// buildNamedOps builds update operations for named-object arrays
// (components, fixVersions). Entries prefixed with - become remove operations.
func buildNamedOps(values []string) []map[string]any {
	ops := make([]map[string]any, 0, len(values))
	for _, v := range values {
		if strings.HasPrefix(v, "-") {
			ops = append(ops, map[string]any{"remove": map[string]any{"name": v[1:]}})
		} else {
			ops = append(ops, map[string]any{"add": map[string]any{"name": v}})
		}
	}
	return ops
}
