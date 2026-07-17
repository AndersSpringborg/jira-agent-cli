package context

import (
	"fmt"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
)

func newUseCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a named context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}
			name := args[0]
			ctx := config.GetContext(cfg, name)
			if ctx == nil {
				return fmt.Errorf("context '%s' not found", name)
			}
			if config.GetProfile(cfg, ctx.Profile) == nil {
				return fmt.Errorf("profile '%s' for context '%s' not found", ctx.Profile, name)
			}

			cfg.ActiveContext = name
			if err := config.Save(cfg); err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			return driver.Message("Active context set to '%s'.", name)
		},
	}
}
