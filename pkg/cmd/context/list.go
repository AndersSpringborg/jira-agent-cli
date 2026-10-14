package context

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List named contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			contexts, err := f.LoadContexts()
			if err != nil {
				return err
			}
			if config.HasLegacyContexts() {
				cmd.PrintErrln("note: contexts in ~/.config/jira-cli/config.yml are ignored; ask the user to recreate them with `sudo jira context set`")
			}

			rows := make([]map[string]any, 0, len(contexts.Contexts))
			for _, name := range contexts.Names() {
				ctx := contexts.Get(name)
				rows = append(rows, map[string]any{
					"name":    name,
					"active":  name == contexts.ActiveContext,
					"profile": ctx.Profile,
					"project": ctx.Project,
				})
			}

			driver := f.DisplayDriverTo(cmd, cmd.OutOrStdout())
			return driver.List("Contexts", []string{"name", "active", "profile", "project"}, rows)
		},
	}
}
