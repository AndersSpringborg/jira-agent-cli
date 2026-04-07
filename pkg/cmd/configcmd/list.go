package configcmd

import (
	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			profiles := config.ListProfiles(cfg)
			rows := make([]map[string]any, 0, len(profiles))
			for _, name := range profiles {
				isDefault := name == cfg.DefaultProfile
				rows = append(rows, map[string]any{
					"name":    name,
					"default": isDefault,
				})
			}

			return driver.List("Profiles", []string{"name", "default"}, rows)
		},
	}
}
