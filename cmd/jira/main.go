package main

import (
	"os"

	"AndersSpringborg/jira-cli/internal/build"
	cmd "AndersSpringborg/jira-cli/pkg/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd(build.Version, build.Date)
	if err := cmd.Execute(rootCmd); err != nil {
		os.Exit(1)
	}
}
