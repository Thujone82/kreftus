//go:build !windows

package main

func setTerminalTitle(title string) {
	// No-op on non-Windows for now. Some terminals support OSC \\x1b]0;title\\x07
}
