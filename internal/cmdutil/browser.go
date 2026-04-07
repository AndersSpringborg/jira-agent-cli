package cmdutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given URL in the user's default browser.
// Returns an error if the browser could not be launched (e.g. headless
// environment). Callers should fall back to printing the URL.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
	return cmd.Start()
}
