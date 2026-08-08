//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	procSetConsoleScreenBufferSize = kernel32.NewProc("SetConsoleScreenBufferSize")
	procSetConsoleWindowInfo       = kernel32.NewProc("SetConsoleWindowInfo")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

// resizeToPreferred sets the console to preferredCols × preferredRows when possible.
func resizeToPreferred() {
	h, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if h != 0 && h != uintptr(syscall.InvalidHandle) {
		cols := int16(preferredCols)
		rows := int16(preferredRows)

		// Shrink the visible window first so the buffer can be resized freely.
		minWin := smallRect{0, 0, 0, 0}
		_, _, _ = procSetConsoleWindowInfo.Call(h, 1, uintptr(unsafe.Pointer(&minWin)))

		// COORD is passed by value as a DWORD: low word = X, high word = Y.
		packed := uintptr(uint32(uint16(cols)) | uint32(uint16(rows))<<16)
		_, _, _ = procSetConsoleScreenBufferSize.Call(h, packed)

		win := smallRect{Left: 0, Top: 0, Right: cols - 1, Bottom: rows - 1}
		_, _, _ = procSetConsoleWindowInfo.Call(h, 1, uintptr(unsafe.Pointer(&win)))
	}

	// Also emit VT resize for Windows Terminal / hosts that ignore Win32 sizing.
	emitVTResize()
}

func emitVTResize() {
	fmt.Printf("\x1b[8;%d;%dt", preferredRows, preferredCols)
}
