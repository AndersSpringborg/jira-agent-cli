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
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(cfg.Contexts))
			for _, name := range config.ListContexts(cfg) {
				ctx := config.GetContext(cfg, name)
				rows = append(rows, map[string]any{
					"name":    name,
					"active":  name == cfg.ActiveContext,
					"profile": ctx.Profile,
					"project": ctx.Project,
				})
			}

			driver := f.DisplayDriver(cmd)
			return driver.List("Contexts", []string{"name", "active", "profile", "project"}, rows)
		},
	}
}
