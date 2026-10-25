//go:build windows

package main

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestMouseInputPacketSize(t *testing.T) {
	got := unsafe.Sizeof(mouseInputPacket{})
	switch unsafe.Sizeof(uintptr(0)) {
	case 8:
		if got != 40 {
			t.Fatalf("INPUT size = %d, want 40", got)
		}
	case 4:
		if got != 28 {
			t.Fatalf("INPUT size = %d, want 28", got)
		}
	default:
		t.Fatalf("unexpected pointer size %d", unsafe.Sizeof(uintptr(0)))
	}
}

func TestScaleToAbsolute(t *testing.T) {
	if got := scaleToAbsolute(0, 1920); got != 0 {
		t.Fatalf("origin %d", got)
	}
	if got := scaleToAbsolute(1919, 1920); got != 65535 {
		t.Fatalf("far edge %d", got)
	}
	mid := scaleToAbsolute(960, 1920)
	if mid < 32000 || mid > 33000 {
		t.Fatalf("mid %d", mid)
	}
}

var (
	hitCallback uintptr
	hitClass    = utf16Ptr("ClickPlaybackHit")
	hitDown     atomic.Int32
	hitUp       atomic.Int32
)

func hitWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case 0x0201: // WM_LBUTTONDOWN
		hitDown.Add(1)
	case 0x0202: // WM_LBUTTONUP
		hitUp.Add(1)
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

func TestPlaybackClickReachesWindow(t *testing.T) {
	runtime.LockOSThread()
	enableDPI()
	if hitCallback == 0 {
		hitCallback = syscall.NewCallback(hitWndProc)
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   hitCallback,
		hInstance:     inst,
		hbrBackground: colorBtnFace,
		lpszClassName: hitClass,
	}
	atom, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)
	if atom == 0 {
		t.Fatal("register hit window")
	}

	const x, y, w, h int32 = 180, 160, 360, 260
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(hitClass)),
		uintptr(unsafe.Pointer(hitClass)),
		uintptr(wsCaption|wsSysMenu|wsVisible),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0, 0, inst, 0,
	)
	if hwnd == 0 {
		t.Fatal("create hit window")
	}
	var before point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&before)))
	t.Cleanup(func() {
		procDestroyWindow.Call(hwnd)
		procSetCursorPos.Call(uintptr(before.x), uintptr(before.y))
		sendAbsolute(before.x, before.y, mouseEventLeftUp, 0)
	})
	procShowWindow.Call(hwnd, swShow)
	procSetForegroundWindow.Call(hwnd)
	pumpMessages(150 * time.Millisecond)

	hitDown.Store(0)
	hitUp.Store(0)
	cx, cy := x+w/2, y+h/2
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = playEvents([]Event{
			{Offset: 0, Kind: Move, X: cx, Y: cy},
			{Offset: 40 * time.Millisecond, Kind: LeftDown, X: cx, Y: cy},
			{Offset: 50 * time.Millisecond, Kind: LeftUp, X: cx, Y: cy},
		}, nil)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && (hitDown.Load() == 0 || hitUp.Load() == 0) {
		pumpMessages(15 * time.Millisecond)
	}
	<-done
	if hitDown.Load() == 0 || hitUp.Load() == 0 {
		var pt point
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		t.Fatalf("click missed window down=%d up=%d cursor=%d,%d want %d,%d", hitDown.Load(), hitUp.Load(), pt.x, pt.y, cx, cy)
	}
}

func pumpMessages(d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		var m winMsg
		r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)
		if r == 0 {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
