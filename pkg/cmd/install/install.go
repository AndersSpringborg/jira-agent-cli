package install

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

const (
	defaultBinDir        = "/usr/local/bin"
	defaultCompletionDir = "/opt/homebrew/etc/bash_completion.d"
)

func NewCmd(root *cobra.Command) *cobra.Command {
	var (
		binDir        string
		completionDir string
		noCompletion  bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install jira binary and shell completions system-wide",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installBinary(binDir); err != nil {
				return err
			}
			if !noCompletion {
				if err := installCompletion(root, completionDir); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&binDir, "bin-dir", defaultBinDir, "Directory to install the binary into")
	cmd.Flags().StringVar(&completionDir, "completion-dir", defaultCompletionDir, "Directory to install bash completions into")
	cmd.Flags().BoolVar(&noCompletion, "no-completion", false, "Skip completion installation")

	return cmd
}

func installBinary(binDir string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	dest := filepath.Join(binDir, "jira")

	if isSameFile(self, dest) {
		fmt.Printf("%s Binary is already installed at %s\n", output.Green("✓"), dest)
		return nil
	}

	fmt.Printf("%s Installing jira to %s...\n", output.Dim("→"), dest)

	if needsSudo(binDir) {
		fmt.Println("  (sudo may prompt for your password)")
		if err := sudoCp(self, dest); err != nil {
			fmt.Printf("%s %s\n", output.Red("✗"), output.Red("Failed to install binary: "+err.Error()))
			return err
		}
	} else {
		data, err := os.ReadFile(self)
		if err != nil {
			return fmt.Errorf("read binary: %w", err)
		}
		if err := os.WriteFile(dest, data, 0o755); err != nil {
			return fmt.Errorf("write binary: %w", err)
		}
	}

	fmt.Printf("%s Binary installed to %s\n", output.Green("✓"), dest)
	return nil
}

func installCompletion(root *cobra.Command, completionDir string) error {
	dest := filepath.Join(completionDir, "jira")
	fmt.Printf("%s Installing bash completions to %s...\n", output.Dim("→"), dest)

	var buf bytes.Buffer
	if err := root.GenBashCompletionV2(&buf, true); err != nil {
		return fmt.Errorf("generate completions: %w", err)
	}

	if needsSudo(completionDir) {
		c := exec.Command("sudo", "tee", dest)
		c.Stdin = &buf
		c.Stdout = nil
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("%s %s\n", output.Red("✗"), output.Red("Failed to install completions: "+err.Error()))
			return err
		}
	} else {
		if err := os.MkdirAll(completionDir, 0o755); err != nil {
			return fmt.Errorf("create completion dir: %w", err)
		}
		if err := os.WriteFile(dest, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write completions: %w", err)
		}
	}

	fmt.Printf("%s Completions installed to %s\n", output.Green("✓"), dest)
	return nil
}

func isSameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}

func needsSudo(dir string) bool {
	f, err := os.CreateTemp(dir, ".jira-write-test-*")
	if err != nil {
		return true
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return false
}

func sudoCp(src, dest string) error {
	c := exec.Command("sudo", "cp", src, dest)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
