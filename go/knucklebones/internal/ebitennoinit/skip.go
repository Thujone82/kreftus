//go:build windows || darwin || (linux && !android) || freebsd || netbsd || openbsd

package noinit

import (
	"os"
	"strings"
)

// shouldSkipEbitenUI is true when this process must not initialize GLFW.
// os.Args is available during init, so -tui can suppress the GUI stack even
// though Ebitengine would otherwise open it before main runs.
func shouldSkipEbitenUI() bool {
	for _, a := range os.Args[1:] {
		switch a {
		case "-tui":
			return true
		case "--":
			return false
		}
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("KNUCKLEBONES_TUI")))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	for _, k := range []string{"SSH_CLIENT", "SSH_CONNECTION", "SSH_TTY"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return nonInteractiveDesktop()
}
