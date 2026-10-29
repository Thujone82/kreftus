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
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
	procSetThreadExecutionState       = kernel32.NewProc("SetThreadExecutionState")
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
	procIsWindowVisible               = user32.NewProc("IsWindowVisible")
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
	procMapVirtualKeyW                = user32.NewProc("MapVirtualKeyW")
	procRegisterHotKey                = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procOpenInputDesktop              = user32.NewProc("OpenInputDesktop")
	procCloseDesktop                  = user32.NewProc("CloseDesktop")
	procCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	procAppendMenuW                   = user32.NewProc("AppendMenuW")
	procTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                   = user32.NewProc("DestroyMenu")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procMoveWindow                    = user32.NewProc("MoveWindow")
	procSetFocus                      = user32.NewProc("SetFocus")
	procCreateFontW                   = gdi32.NewProc("CreateFontW")
	procDeleteObject                  = gdi32.NewProc("DeleteObject")
	procGetDeviceCaps                 = gdi32.NewProc("GetDeviceCaps")
	procTimeBeginPeriod               = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod                 = winmm.NewProc("timeEndPeriod")
	procShellNotifyIconW              = shell32.NewProc("Shell_NotifyIconW")
	procRegOpenKeyExW                 = advapi32.NewProc("RegOpenKeyExW")
	procRegSetValueExW                = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW               = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey                   = advapi32.NewProc("RegCloseKey")
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

// isDesktopUnlocked checks whether the interactive input desktop is available
// (returns false when the workstation is locked or on the UAC secure desktop).
func isDesktopUnlocked() bool {
	if procOpenInputDesktop.Find() != nil {
		return true
	}
	h, _, _ := procOpenInputDesktop.Call(0, 0, 0x0100) // DESKTOP_SWITCHDESKTOP
	if h == 0 {
		return false
	}
	procCloseDesktop.Call(h)
	return true
}

// setKeepAwake toggles ES_SYSTEM_REQUIRED on the calling thread so Windows does
// not enter sleep while active schedules are waiting.
func setKeepAwake(on bool) {
	const (
		esContinuous     = 0x80000000
		esSystemRequired = 0x00000001
	)
	flags := uintptr(esContinuous)
	if on {
		flags |= esSystemRequired
	}
	procSetThreadExecutionState.Call(flags)
}

// setStartWithWindows adds or removes Click in HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
func setStartWithWindows(enable bool, exePath string) error {
	const (
		hkeyCurrentUser = 0x80000001
		keySetValue     = 0x0002
		regSz           = 1
	)
	subKey := utf16Ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKey)),
		0,
		keySetValue,
		uintptr(unsafe.Pointer(&hKey)),
	)
	runtime.KeepAlive(subKey)
	if r != 0 {
		return syscall.Errno(r)
	}
	defer procRegCloseKey.Call(hKey)

	valName := utf16Ptr("ClickMacroApp")
	defer runtime.KeepAlive(valName)
	if !enable {
		procRegDeleteValueW.Call(hKey, uintptr(unsafe.Pointer(valName)))
		return nil
	}
	quoted := `"` + exePath + `"`
	u16, err := syscall.UTF16FromString(quoted)
	if err != nil {
		return err
	}
	r2, _, _ := procRegSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valName)),
		0,
		regSz,
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)*2),
	)
	runtime.KeepAlive(u16)
	if r2 != 0 {
		return syscall.Errno(r2)
	}
	return nil
}
