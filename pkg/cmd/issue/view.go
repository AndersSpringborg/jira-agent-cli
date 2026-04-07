package issue

import (
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		fields    string
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

			var fieldList []string
			if fields != "" {
				for _, fld := range strings.Split(fields, ",") {
					fld = strings.TrimSpace(fld)
					if fld != "" {
						fieldList = append(fieldList, fld)
					}
				}
			}

			// Request comments if the user asked for them.
			if comments > 0 {
				fieldList = append(fieldList, "comment")
			}

			data, err := client.GetIssue(issueKey, fieldList)
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
	cmd.Flags().IntVar(&comments, "comments", 0, "Number of recent comments to display")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Print raw JSON")

	return cmd
}
