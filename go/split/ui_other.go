//go:build !windows

package main

import "fmt"

func runGUI(_ string) error {
	return fmt.Errorf("GUI mode is only supported on Windows; run with CLI flags (e.g. -help)")
}

func showMessageBoxError(_, _ string) bool { return false }
