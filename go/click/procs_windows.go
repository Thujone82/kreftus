//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	procGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procPeekMessageW                  = user32.NewProc("PeekMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procIsDialogMessageW              = user32.NewProc("IsDialogMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState              = user32.NewProc("GetAsyncKeyState")
	procSendMessageW                  = user32.NewProc("SendMessageW")
	procSetWindowTextW                = user32.NewProc("SetWindowTextW")
	procGetWindowTextW                = user32.NewProc("GetWindowTextW")
	procEnableWindow                  = user32.NewProc("EnableWindow")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procMessageBoxW                   = user32.NewProc("MessageBoxW")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procLoadIconW                     = user32.NewProc("LoadIconW")
	procLoadImageW                    = user32.NewProc("LoadImageW")
	procAdjustWindowRectEx            = user32.NewProc("AdjustWindowRectEx")
	procGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	procSetTimer                      = user32.NewProc("SetTimer")
	procKillTimer                     = user32.NewProc("KillTimer")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procSetCursorPos                  = user32.NewProc("SetCursorPos")
	procSendInput                     = user32.NewProc("SendInput")
	procRegisterHotKey                = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procMoveWindow                    = user32.NewProc("MoveWindow")
	procSetFocus                      = user32.NewProc("SetFocus")
	procCreateFontW                   = gdi32.NewProc("CreateFontW")
	procDeleteObject                  = gdi32.NewProc("DeleteObject")
	procGetDeviceCaps                 = gdi32.NewProc("GetDeviceCaps")
	procTimeBeginPeriod               = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod                 = winmm.NewProc("timeEndPeriod")
)

type rect struct {
	left, top, right, bottom int32
}

type point struct {
	x, y int32
}

type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		// Constant UI strings are valid UTF-8. A failure here is a programming error.
		panic(err)
	}
	return p
}

func setText(hwnd uintptr, s string) {
	if hwnd == 0 {
		return
	}
	p := utf16Ptr(s)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func getText(hwnd uintptr) string {
	var buf [64]uint16
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n > uintptr(len(buf)) {
		n = uintptr(len(buf))
	}
	return syscall.UTF16ToString(buf[:n])
}

func enableWindow(hwnd uintptr, on bool) {
	if hwnd == 0 {
		return
	}
	v := uintptr(0)
	if on {
		v = 1
	}
	procEnableWindow.Call(hwnd, v)
}

func messageBox(owner uintptr, title, text string) {
	t := utf16Ptr(title)
	b := utf16Ptr(text)
	const mbIconWarning = 0x30
	procMessageBoxW.Call(owner, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), mbIconWarning)
	runtime.KeepAlive(t)
	runtime.KeepAlive(b)
}

func postMessage(hwnd uintptr, msg, wParam, lParam uintptr) {
	if hwnd == 0 {
		return
	}
	procPostMessageW.Call(hwnd, msg, wParam, lParam)
}
