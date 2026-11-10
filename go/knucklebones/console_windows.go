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
	swShow          = 5
	stdInputHandle  = ^uintptr(9)  // STD_INPUT_HANDLE (-10)
	stdErrorHandle  = ^uintptr(11) // STD_ERROR_HANDLE (-12)
)

func consoleWindow() uintptr {
	h, _, _ := procGetConsoleWindow.Call()
	return h
}

// launchedFromTerminal is false for a double-click: Explorer gives the
// process a console of its own, which Ebiten's hideconsole package frees
// before main runs. A shell sharing the console makes it true.
func launchedFromTerminal() bool {
	if consoleWindow() == 0 {
		return false
	}
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	return n != 1
}

// detachConsole closes a console window nobody else is using.
func detachConsole() {
	_, _, _ = procFreeConsole.Call()
}

// ensureConsole gives the TUI a visible console, opening a new one when
// the double-click console was already freed.
func ensureConsole() {
	if h := consoleWindow(); h != 0 {
		_, _, _ = procShowWindow.Call(h, swShow)
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
