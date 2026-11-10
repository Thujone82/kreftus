package main

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// remoteSSH is true when the process was started through an SSH session.
// Windows OpenSSH often has no Win32 console HWND even though stdin is a PTY.
func remoteSSH() bool {
	for _, k := range []string{"SSH_CLIENT", "SSH_CONNECTION", "SSH_TTY"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// forceTUIEnv lets operators force the terminal UI without flags.
func forceTUIEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("KNUCKLEBONES_TUI")))
	return v == "1" || v == "true" || v == "yes"
}

func stdioIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) || term.IsTerminal(int(os.Stdout.Fd()))
}

// guiBlockedSession is a headless/SSH session where creating an Ebiten window
// fails with ERROR_REQUIRES_INTERACTIVE_WINDOWSTATION.
func guiBlockedSession() bool {
	return remoteSSH() || forceTUIEnv() || sshLikeConsole()
}

// needShowConsole is whether ShowWindow is safe for the attached console.
// SSH sessions sit on a non-interactive window station, so ShowWindow fails.
func needShowConsole(hasWindow, ssh bool) bool {
	return hasWindow && !ssh
}

// needAllocConsole is whether the TUI should open a fresh Win32 console.
// That is only for local double-click -tui after the console was freed.
// SSH/PTY already have stdio and must not call AllocConsole.
func needAllocConsole(hasWindow, ssh, pty bool) bool {
	return !hasWindow && !ssh && !pty
}

// needWin32ConsoleUI is whether font and buffer-size Win32 calls are useful.
// Without a console window (SSH ConPTY), stick to VT sequences only.
func needWin32ConsoleUI(hasWindow, ssh bool) bool {
	return hasWindow && !ssh
}
