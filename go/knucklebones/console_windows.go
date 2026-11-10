//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	procAllocConsole          = kernel32.NewProc("AllocConsole")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
	procShowWindow            = syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
)

const (
	swShow         = 5
	stdInputHandle = ^uintptr(9)  // STD_INPUT_HANDLE (-10)
	stdErrorHandle = ^uintptr(11) // STD_ERROR_HANDLE (-12)
)

func consoleWindow() uintptr {
	h, _, _ := procGetConsoleWindow.Call()
	return h
}

// hasConsoleInput is true when stdin is a real console or ConPTY handle.
// Windows OpenSSH often has this without a GetConsoleWindow HWND.
func hasConsoleInput() bool {
	h, _, _ := procGetStdHandle.Call(stdInputHandle)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return false
	}
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

// sshLikeConsole detects Windows OpenSSH ConPTY: console I/O works, but there
// is no interactive desktop HWND for a GUI window.
func sshLikeConsole() bool {
	return consoleWindow() == 0 && hasConsoleInput()
}

// launchedFromTerminal is false for a double-click: Explorer gives the
// process a console of its own. A shell, SSH session, or ConPTY makes it true.
func launchedFromTerminal() bool {
	if remoteSSH() || forceTUIEnv() || sshLikeConsole() {
		return true
	}
	if hwnd := consoleWindow(); hwnd != 0 {
		var pids [2]uint32
		n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
		// Sole console owner ⇒ Explorer (or similar) double-click.
		if n == 1 {
			return false
		}
		if n > 1 {
			return true
		}
	}
	// No Win32 console window: pipes or a detached process.
	return stdioIsTerminal()
}

// detachConsole closes a console window nobody else is using.
func detachConsole() {
	_, _, _ = procFreeConsole.Call()
}

// ensureConsole gives the TUI a visible console after a double-click -tui
// launch when no console remains. It must not AllocConsole or ShowWindow
// over SSH: those need an interactive window station.
func ensureConsole() {
	hasWindow := consoleWindow() != 0
	ssh := remoteSSH()
	if needShowConsole(hasWindow, ssh) {
		_, _, _ = procShowWindow.Call(consoleWindow(), swShow)
		return
	}
	pty := stdioIsTerminal() || hasConsoleInput()
	if !needAllocConsole(hasWindow, ssh, pty) {
		return
	}
	if r, _, _ := procAllocConsole.Call(); r == 0 {
		return
	}
	reopen := func(id uintptr, name string) *os.File {
		h, _, _ := procGetStdHandle.Call(id)
		return os.NewFile(h, name)
	}
	os.Stdin = reopen(stdInputHandle, "/dev/stdin")
	os.Stdout = reopen(stdOutputHandle, "/dev/stdout")
	os.Stderr = reopen(stdErrorHandle, "/dev/stderr")
}
