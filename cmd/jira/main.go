package main

import (
	"fmt"
	"os"

	"AndersSpringborg/jira-cli/internal/build"
	cmd "AndersSpringborg/jira-cli/pkg/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd(build.Version, build.Date)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
