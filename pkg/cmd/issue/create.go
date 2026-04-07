package issue

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		project     string
		summary     string
		issueType   string
		description string
		priority    string
		labels      []string
		parent      string
		fixVersions []string
		components  []string
		noInput     bool
		rawOutput   bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				profile, err := f.LoadProfile()
				if err != nil {
					return err
				}
				if profile.Context != nil && profile.Context.Project != "" {
					project = profile.Context.Project
				}
			}
			if project == "" {
				return fmt.Errorf("--project is required (or set via `jira context set --project`)")
			}
			if summary == "" {
				return fmt.Errorf("--summary is required")
			}
			if issueType == "" {
				issueType = "Task"
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			data, err := client.CreateIssue(project, summary, issueType, description, priority, labels, parent, components, fixVersions)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)

			if rawOutput {
				return driver.Raw(data)
			}

			return driver.Message("Created issue: %v", data["key"])
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key or ID")
	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Issue summary")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "Issue type (default: Task)")
	cmd.Flags().StringVarP(&description, "body", "b", "", "Issue description")
	cmd.Flags().StringVarP(&priority, "priority", "y", "", "Issue priority")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Label (repeatable)")
	cmd.Flags().StringVarP(&parent, "parent", "P", "", "Parent issue key (epic)")
	cmd.Flags().StringSliceVarP(&components, "component", "C", nil, "Component (repeatable)")
	cmd.Flags().StringSliceVar(&fixVersions, "fix-version", nil, "Fix version (repeatable)")
	cmd.Flags().BoolVar(&noInput, "no-input", false, "Disable interactive prompt")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Print raw JSON")

	return cmd
}
