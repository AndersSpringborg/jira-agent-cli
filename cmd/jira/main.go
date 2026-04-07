package main

import (
	"fmt"
	"os"

	cmd "AndersSpringborg/jira-cli/pkg/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
