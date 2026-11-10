//go:build windows

package noinit

import (
	"syscall"
	"unsafe"
)

var (
	modUser32                      = syscall.NewLazyDLL("user32.dll")
	procGetProcessWindowStation    = modUser32.NewProc("GetProcessWindowStation")
	procGetUserObjectInformationW  = modUser32.NewProc("GetUserObjectInformationW")
)

const uguoiName = 2 // UOI_NAME

// nonInteractiveDesktop reports a Session 0 / non-WinSta0 process, where
// CreateWindowEx fails with ERROR_REQUIRES_INTERACTIVE_WINDOWSTATION.
func nonInteractiveDesktop() bool {
	hwinsta, _, _ := procGetProcessWindowStation.Call()
	if hwinsta == 0 {
		return true
	}
	var needed uint32
	var buf [256]uint16
	r, _, _ := procGetUserObjectInformationW.Call(
		hwinsta,
		uintptr(uguoiName),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)*2),
		uintptr(unsafe.Pointer(&needed)),
	)
	if r == 0 {
		return true
	}
	name := syscall.UTF16ToString(buf[:])
	return name != "WinSta0"
}
