//go:build !windows

package main

// launchedFromTerminal is false when a file manager or desktop launcher
// started the program without a terminal attached.
func launchedFromTerminal() bool {
	return remoteSSH() || forceTUIEnv() || stdioIsTerminal()
}

func sshLikeConsole() bool { return false }

func detachConsole() {}

func ensureConsole() {}
