//go:build darwin || (linux && !android) || freebsd || netbsd || openbsd

package noinit

import "os"

// nonInteractiveDesktop is true when no graphical display is available.
func nonInteractiveDesktop() bool {
	return os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == ""
}
