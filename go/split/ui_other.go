//go:build !windows && !darwin && !linux

package main

import "fmt"

func runGUI(_ string) error {
	return fmt.Errorf("GUI mode is not supported on this platform; run with CLI flags (e.g. -help)")
}

func showMessageBoxError(_, _ string) bool { return false }
