package context

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// openForChange returns the context store and its contents, failing first —
// before any other validation — when the caller lacks sudo, so an agent sees
// the guidance immediately.
func openForChange(f *cmdutil.Factory, cmd *cobra.Command, args []string) (*config.ContextStore, *config.ContextFile, error) {
	store, err := f.ContextStore()
	if err != nil {
		return nil, nil, err
	}
	if err := store.RequirePrivilege(); err != nil {
		return nil, nil, fmt.Errorf(`%w

The context limits what AI agents can see and change in Jira, so only the user may change it.
AI agents: Ask the user to run this command themselves. Never run it yourself with sudo,
and never try to bypass or weaken this protection (editing context files, changing file
permissions or ownership, or reusing cached sudo credentials).

Ask the user to run this command:

  %s`, err, privilegedCommand(cmd, args))
	}
	contexts, err := store.Load()
	if err != nil {
		return nil, nil, err
	}
	return store, contexts, nil
}

// privilegedCommand reconstructs the invocation, prefixed for elevation.
func privilegedCommand(cmd *cobra.Command, args []string) string {
	parts := []string{cmd.CommandPath()}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		name := "--" + flag.Name
		switch value := flag.Value.(type) {
		case pflag.SliceValue:
			for _, item := range value.GetSlice() {
				parts = append(parts, name, shellQuote(item))
			}
		default:
			if flag.Value.Type() == "bool" {
				if flag.Value.String() == "true" {
					parts = append(parts, name)
				} else {
					parts = append(parts, name+"=false")
				}
				return
			}
			parts = append(parts, name, shellQuote(flag.Value.String()))
		}
	})
	command := strings.Join(parts, " ")
	if runtime.GOOS == "windows" {
		return command + "    (from a terminal opened with \"Run as administrator\")"
	}
	return "sudo " + command
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_./:=,@%+-]+$`)

func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
