//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procPostThreadMessageW            = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadID            = kernel32.NewProc("GetCurrentThreadId")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procUpdateWindow                  = user32.NewProc("UpdateWindow")
	procSetFocus                      = user32.NewProc("SetFocus")
	procGetClientRect                 = user32.NewProc("GetClientRect")
	procGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	procLoadImageW                    = user32.NewProc("LoadImageW")
	procLoadIconW                     = user32.NewProc("LoadIconW")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procSendMessageW                  = user32.NewProc("SendMessageW")
	procSetWindowTextW                = user32.NewProc("SetWindowTextW")
	procMessageBoxW                   = user32.NewProc("MessageBoxW")
	procChangeWindowMessageFilterEx   = user32.NewProc("ChangeWindowMessageFilterEx")

	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
	procDragQueryFileW  = shell32.NewProc("DragQueryFileW")
	procDragFinish      = shell32.NewProc("DragFinish")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wmDestroy          = 0x0002
	wmSize             = 0x0005
	wmActivate         = 0x0006
	wmClose            = 0x0010
	wmGetMinMaxInfo    = 0x0024
	wmCopyGlobalData   = 0x0049
	wmSetIcon          = 0x0080
	wmNCLButtonDown    = 0x00A1
	wmMoving           = 0x0216
	wmMove             = 0x0003
	wmDropFiles        = 0x0233
	wmApp              = 0x8000
	wmQuit             = 0x0012
	waInactive         = 0

	swShow         = 5
	smCxScreen     = 0
	smCyScreen     = 1
	smCxIcon       = 11
	smCyIcon       = 12
	smCxSmIcon     = 49
	smCySmIcon     = 50
	imageIcon      = 1
	lrShared       = 0x8000
	iconSmall      = 0
	iconBig        = 1
	idcArrow       = 32512
	idiApplication = 32512

	mbYesNo       = 0x00000004
	mbIconWarning = 0x00000030
	mbIconError   = 0x00000010
	idYes         = 6

	msgfltAllow = 1
	dwmDarkMode = 20 // DWMWA_USE_IMMERSIVE_DARK_MODE
)

type winPoint struct {
	X, Y int32
}

type winRect struct {
	Left, Top, Right, Bottom int32
}

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      winPoint
}

type minMaxInfo struct {
	PtReserved     winPoint
	PtMaxSize      winPoint
	PtMaxPosition  winPoint
	PtMinTrackSize winPoint
	PtMaxTrackSize winPoint
}

type wndClassExW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

func enablePerMonitorDPI() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		r, _, _ := procSetProcessDpiAwarenessContext.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
		if r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
}

func setWindowDarkMode(hwnd uintptr, dark bool) {
	if hwnd == 0 || procDwmSetWindowAttribute.Find() != nil {
		return
	}
	var val int32
	if dark {
		val = 1
	}
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, dwmDarkMode, uintptr(unsafe.Pointer(&val)), unsafe.Sizeof(val))
}

func showMessageBoxError(title, text string) bool {
	t := utf16Ptr(title)
	b := utf16Ptr(text)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), mbIconError)
	runtime.KeepAlive(t)
	runtime.KeepAlive(b)
	return true
}

func confirmOverwriteDialog(owner uintptr, title, text string) bool {
	t := utf16Ptr(title)
	b := utf16Ptr(text)
	ret, _, _ := procMessageBoxW.Call(owner, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), mbYesNo|mbIconWarning)
	runtime.KeepAlive(t)
	runtime.KeepAlive(b)
	return ret == idYes
}
