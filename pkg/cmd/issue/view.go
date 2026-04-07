package issue

import (
	"fmt"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		fields    string
		comments  int
		plain     bool
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

			var fieldList []string
			if fields != "" {
				for _, fld := range strings.Split(fields, ",") {
					fld = strings.TrimSpace(fld)
					if fld != "" {
						fieldList = append(fieldList, fld)
					}
				}
			}

			data, err := client.GetIssue(issueKey, fieldList)
			if err != nil {
				return err
			}

			if rawOutput {
				return output.JSON(data)
			}

			f2, _ := data["fields"].(map[string]any)
			row := map[string]any{
				"key": data["key"],
			}
			if f2 != nil {
				row["summary"] = f2["summary"]
				row["status"] = f2["status"]
				row["assignee"] = f2["assignee"]
				row["priority"] = f2["priority"]
				row["issuetype"] = f2["issuetype"]
				row["reporter"] = f2["reporter"]
				row["created"] = f2["created"]
				row["updated"] = f2["updated"]
			}

			if plain {
				cols := []string{"key", "summary", "status", "assignee", "priority", "issuetype"}
				output.TableWithOptions([]map[string]any{row}, cols, "Issue", output.TableOptions{Plain: true})
			} else {
				cols := []string{"key", "summary", "status", "assignee", "priority", "issuetype"}
				output.Table([]map[string]any{row}, cols, "Issue")
			}

			if f2 != nil {
				if desc := f2["description"]; desc != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "\nDescription:\n%v\n", desc)
				}

				if comments > 0 {
					if commentField, ok := f2["comment"].(map[string]any); ok {
						if commentList, ok := commentField["comments"].([]any); ok {
							start := len(commentList) - comments
							if start < 0 {
								start = 0
							}
							fmt.Fprintf(cmd.OutOrStdout(), "\nComments:\n")
							for _, c := range commentList[start:] {
								cm, ok := c.(map[string]any)
								if !ok {
									continue
								}
								author := ""
								if a, ok := cm["author"].(map[string]any); ok {
									author = output.FormatValue(a)
								}
								fmt.Fprintf(cmd.OutOrStdout(), "\n--- %s ---\n%v\n", author, cm["body"])
							}
						}
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to fetch")
	cmd.Flags().IntVar(&comments, "comments", 0, "Number of recent comments to display")
	cmd.Flags().BoolVar(&plain, "plain", false, "Plain output")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Print raw JSON")

	return cmd
}
