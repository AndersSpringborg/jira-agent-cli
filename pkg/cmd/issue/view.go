package issue

import (
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func parseViewFields(commaSeparated string, repeatable []string) []string {
	fields := make([]string, 0, len(repeatable))
	if commaSeparated != "" {
		for _, fld := range strings.Split(commaSeparated, ",") {
			fld = strings.TrimSpace(fld)
			if fld != "" {
				fields = append(fields, fld)
			}
		}
	}
	for _, fld := range repeatable {
		fld = strings.TrimSpace(fld)
		if fld != "" {
			fields = append(fields, fld)
		}
	}
	return fields
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		fields    string
		fieldList []string
		comments  int
		rawOutput bool
	)

	cmd := &cobra.Command{
		Use:     "view <issue-key>",
		Aliases: []string{"get"},
		Short:   "View issue details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			requestedFields := parseViewFields(fields, fieldList)

			// Request comments if the user asked for them.
			if comments > 0 {
				requestedFields = append(requestedFields, "comment")
			}

			data, err := client.GetIssue(issueKey, requestedFields)
			if err != nil {
				return err
			}

			if rawOutput {
				return driver.Raw(data)
			}

			// Build a display-friendly map from the Jira issue structure.
			issueData := map[string]any{
				"key": data["key"],
			}

			flds, _ := data["fields"].(map[string]any)
			if flds != nil {
				// Copy fields and trim comments to the requested count.
				displayFields := map[string]any{}
				for k, v := range flds {
					displayFields[k] = v
				}

				if comments > 0 {
					if commentField, ok := displayFields["comment"].(map[string]any); ok {
						if commentList, ok := commentField["comments"].([]any); ok {
							start := len(commentList) - comments
							if start < 0 {
								start = 0
							}
							displayFields["comment"] = map[string]any{
								"total":    commentField["total"],
								"comments": commentList[start:],
							}
						}
					}
				}

				issueData["fields"] = displayFields
			}

			return driver.Item("Issue", issueData)
		},
	}

	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to fetch")
	cmd.Flags().StringArrayVarP(&fieldList, "field", "F", nil, "Field to fetch (repeatable)")
	cmd.Flags().IntVar(&comments, "comments", 0, "Number of recent comments to display")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Print raw JSON")

	return cmd
}
