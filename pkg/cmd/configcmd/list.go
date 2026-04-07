package configcmd

import (
	"fmt"

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

			for _, name := range config.ListProfiles(cfg) {
				marker := " "
				if name == cfg.DefaultProfile {
					marker = "*"
				}
				fmt.Printf("%s %s\n", marker, name)
			}
			return nil
		},
	}
}
