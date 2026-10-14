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
			store, contexts, err := openForChange(f, cmd, args)
			if err != nil {
				return err
			}
			cfg, err := f.LoadConfig()
			if err != nil {
				return err
			}
			name := args[0]
			ctx := contexts.Get(name)
			if ctx == nil {
				return fmt.Errorf("context '%s' not found", name)
			}
			if config.GetProfile(cfg, ctx.Profile) == nil {
				return fmt.Errorf("profile '%s' for context '%s' not found", ctx.Profile, name)
			}

			contexts.ActiveContext = name
			if err := store.Save(contexts); err != nil {
				return err
			}

			driver := f.DisplayDriverTo(cmd, cmd.OutOrStdout())
			return driver.Message("Active context set to '%s'.", name)
		},
	}
}
