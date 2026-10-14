package boundary

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

// openForChange returns the boundary store, failing first — before any other
// validation — when the caller lacks sudo, so an agent sees the guidance
// immediately.
func openForChange(f *cmdutil.Factory, cmd *cobra.Command, args []string) (*config.BoundaryStore, error) {
	store, err := f.BoundaryStore()
	if err != nil {
		return nil, err
	}
	if err := store.RequirePrivilege(); err != nil {
		return nil, fmt.Errorf(`%w

The boundary limits which Jira issues AI agents can see and change, so only the user may change it.
AI agents: Ask the user to run this command themselves. Never run it yourself with sudo,
and never try to bypass or weaken this protection (editing boundary files, changing file
permissions or ownership, or reusing cached sudo credentials).

Ask the user to run this command:

  %s`, err, privilegedCommand(cmd, args))
	}
	return store, nil
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
