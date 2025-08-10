//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTitleW = modkernel32.NewProc("SetConsoleTitleW")
)

func setTerminalTitle(title string) {
	utf16, _ := syscall.UTF16PtrFromString(title)
	procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(utf16)))
}
