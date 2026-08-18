package cli

import (
	"os/exec"
	"runtime"
)

// openBrowser opens url in the system's default browser. Errors are the
// caller's to decide whether to surface - `alaws ui` treats this as
// best-effort, since the server is already running and usable even if the
// browser doesn't open automatically.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
