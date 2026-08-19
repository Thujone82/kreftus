//go:build windows

package main

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32                      = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState        = user32.NewProc("GetAsyncKeyState")
	procSetWindowsHookExW       = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx          = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx     = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW             = user32.NewProc("GetMessageW")
	procPostThreadMessageW      = user32.NewProc("PostThreadMessageW")
	shiftKernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandleW        = shiftKernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId      = shiftKernel32.NewProc("GetCurrentThreadId")
)

const (
	vkShift        = 0x10
	whKeyboardLL   = 13
	wmKeyDown      = 0x0100
	wmSysKeyDown   = 0x0104
	wmQuit         = 0x0012
	llkhfExtended  = 0x01
)

type kbdllhookstruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type winmsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

var (
	keyboardLLCallback uintptr
	hookHandle         uintptr
	hookThreadID       uint32
	originMu           sync.Mutex
	originKnown        bool
	lastExtended       bool
)

func shiftPollable() bool { return true }

func shiftHeld() bool {
	r, _, _ := procGetAsyncKeyState.Call(vkShift)
	return r&0x8000 != 0
}

func lowLevelKeyboardProc(nCode int, wParam, lParam uintptr) uintptr {
	if nCode >= 0 && (wParam == wmKeyDown || wParam == wmSysKeyDown) {
		k := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		originMu.Lock()
		originKnown = true
		lastExtended = k.flags&llkhfExtended != 0
		originMu.Unlock()
	}
	r, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return r
}

func startShiftWatch() {
	keyboardLLCallback = syscall.NewCallback(lowLevelKeyboardProc)
	go func() {
		mod, _, _ := procGetModuleHandleW.Call(0)
		h, _, _ := procSetWindowsHookExW.Call(whKeyboardLL, keyboardLLCallback, mod, 0)
		if h == 0 {
			return
		}
		hookHandle = h
		tid, _, _ := procGetCurrentThreadId.Call()
		hookThreadID = uint32(tid)
		var m winmsg
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
		}
	}()
}

func stopShiftWatch() {
	if hookHandle != 0 {
		_, _, _ = procUnhookWindowsHookEx.Call(hookHandle)
		hookHandle = 0
	}
	if hookThreadID != 0 {
		_, _, _ = procPostThreadMessageW.Call(uintptr(hookThreadID), wmQuit, 0, 0)
		hookThreadID = 0
	}
}

// keypadOrigin is true when the last physical key was a non-extended keypad
// key (NumLock+Shift 2/4/6/8). Dedicated arrows are extended and return false.
func keypadOrigin() bool {
	originMu.Lock()
	defer originMu.Unlock()
	if !originKnown {
		return true
	}
	return !lastExtended
}
