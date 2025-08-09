//go:build !windows

package main

import "fmt"

// On non-Windows platforms these are simple no-ops or direct prints.

// non-Windows placeholder used by main
func configureWindowsConsole() {}

func writeConsole(s string) { fmt.Print(s) }
