//go:build !windows

package main

import (
	"os"

	"golang.org/x/term"
)

// launchedFromTerminal is false when a file manager or desktop launcher
// started the program without a terminal attached.
func launchedFromTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func detachConsole() {}

func ensureConsole() {}
