//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

const (
	wmTrayCallback = 0x0400 + 50 // WM_USER + 50

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifInfo    = 0x00000010

	wmLButtonUp     = 0x0202
	wmLButtonDblClk = 0x0203
	wmRButtonUp     = 0x0205

	mfString     = 0x00000000
	mfSeparator  = 0x00000800
	tpmReturnCmd = 0x0100
	tpmRightBtn  = 0x0002

	trayMenuShow   = 1001
	trayMenuRecord = 1002
	trayMenuAbort  = 1003
	trayMenuExit   = 1004
)

// notifyIconDataW matches NOTIFYICONDATAW (V2 size on Windows Vista/7/10/11).
type notifyIconDataW struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

type trayIcon struct {
	hwnd  uintptr
	icon  uintptr
	added bool
}

func (t *trayIcon) init(hwnd, icon uintptr, tip string) {
	t.hwnd = hwnd
	t.icon = icon
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = hwnd
	nid.uID = 1
	nid.uFlags = nifMessage | nifIcon | nifTip
	nid.uCallbackMessage = wmTrayCallback
	nid.hIcon = icon
	copyUTF16(nid.szTip[:], tip)

	r, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	runtime.KeepAlive(&nid)
	t.added = r != 0
}

func (t *trayIcon) setTooltip(tip string) {
	if !t.added {
		return
	}
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = t.hwnd
	nid.uID = 1
	nid.uFlags = nifTip
	copyUTF16(nid.szTip[:], tip)
	procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	runtime.KeepAlive(&nid)
}

func (t *trayIcon) showBalloon(title, body string) {
	if !t.added {
		return
	}
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = t.hwnd
	nid.uID = 1
	nid.uFlags = nifInfo
	copyUTF16(nid.szInfoTitle[:], title)
	copyUTF16(nid.szInfo[:], body)
	nid.dwInfoFlags = 0x00000001 // NIIF_INFO
	procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	runtime.KeepAlive(&nid)
}

func (t *trayIcon) remove() {
	if !t.added {
		return
	}
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = t.hwnd
	nid.uID = 1
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	runtime.KeepAlive(&nid)
	t.added = false
}

func showTrayPopupMenu(hwnd uintptr, isRecording bool) uint32 {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return 0
	}
	defer procDestroyMenu.Call(menu)

	appendMenuStr(menu, trayMenuShow, "Open Click")
	recLabel := "Start Recording"
	if isRecording {
		recLabel = "Stop Recording"
	}
	appendMenuStr(menu, trayMenuRecord, recLabel)
	appendMenuStr(menu, trayMenuAbort, "Abort Countdown / Playback")
	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	appendMenuStr(menu, trayMenuExit, "Exit Click")

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmReturnCmd|tpmRightBtn,
		uintptr(pt.x),
		uintptr(pt.y),
		0,
		hwnd,
		0,
	)
	return uint32(cmd)
}

func appendMenuStr(menu uintptr, id uintptr, label string) {
	p := utf16Ptr(label)
	procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func copyUTF16(dst []uint16, s string) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	if len(u) >= len(dst) {
		copy(dst, u[:len(dst)-1])
		dst[len(dst)-1] = 0
		return
	}
	copy(dst, u)
}
