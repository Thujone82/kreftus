//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Recording polls the cursor and mouse buttons from a normal goroutine.
// A low-level hook, and raw input handled inside the window procedure, both
// stall this process: Windows calls back into Go from inside a message-loop
// syscall, and the window is marked "Not Responding".

const (
	vkLButton = 0x01
	vkRButton = 0x02
	vkMButton = 0x04
)

type recorder struct {
	started time.Time
	stopCh  chan struct{}
	doneCh  chan struct{}
}

var (
	recMu      sync.Mutex
	rawSlots   [maxRecordEvents]RawEvent
	rawN       int
	recOn      bool
	recStart   time.Time
	haveCursor bool
	curX, curY int32
	winL       int32
	winT       int32
	winR       int32
	winB       int32
)

func (r *recorder) start(hwnd uintptr) error {
	recMu.Lock()
	if recOn {
		recMu.Unlock()
		return fmt.Errorf("recording is already active")
	}
	rawN = 0
	haveCursor = false
	recStart = time.Now()
	r.started = recStart
	recOn = true
	recMu.Unlock()
	updateRecordedWindow(hwnd)

	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	go pollMouse(r.stopCh, r.doneCh)
	return nil
}

func updateRecordedWindow(hwnd uintptr) {
	var rc rect
	if hwnd != 0 {
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	}
	recMu.Lock()
	winL, winT, winR, winB = rc.left, rc.top, rc.right, rc.bottom
	recMu.Unlock()
}

func (r *recorder) stop() []RawEvent {
	if r.stopCh != nil {
		select {
		case <-r.stopCh:
		default:
			close(r.stopCh)
		}
		select {
		case <-r.doneCh:
		case <-time.After(300 * time.Millisecond):
		}
		r.stopCh = nil
		r.doneCh = nil
	}
	recMu.Lock()
	defer recMu.Unlock()
	recOn = false
	n := rawN
	if n > maxRecordEvents {
		n = maxRecordEvents
	}
	if n < 0 {
		n = 0
	}
	out := make([]RawEvent, n)
	copy(out, rawSlots[:n])
	rawN = 0
	haveCursor = false
	recStart = time.Time{}
	r.started = time.Time{}
	return out
}

func (r *recorder) snapshot() (n int, length, elapsed time.Duration, on bool) {
	recMu.Lock()
	defer recMu.Unlock()
	on = recOn
	n = rawN
	if n > maxRecordEvents {
		n = maxRecordEvents
	}
	if !recStart.IsZero() {
		elapsed = time.Since(recStart)
	}
	if n <= 0 {
		return 0, 0, elapsed, on
	}
	length = time.Duration(rawSlots[n-1].Tick-rawSlots[0].Tick) * time.Millisecond
	return n, length, elapsed, on
}

func pollMouse(stop <-chan struct{}, done chan<- struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	procTimeBeginPeriod.Call(1)
	defer procTimeEndPeriod.Call(1)
	// Drop the "pressed since last call" bit from before recording started.
	readKey(vkLButton)
	readKey(vkRButton)
	readKey(vkMButton)
	var left, right, middle bool
	ticker := time.NewTicker(4 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			left, right, middle = sampleMouse(left, right, middle)
		}
	}
}

func sampleMouse(left, right, middle bool) (bool, bool, bool) {
	var pt point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return left, right, middle
	}
	l := readKey(vkLButton)
	r := readKey(vkRButton)
	m := readKey(vkMButton)

	recMu.Lock()
	defer recMu.Unlock()
	if !recOn {
		return left, right, middle
	}
	tick := uint32(time.Since(recStart) / time.Millisecond)
	curX, curY = pt.x, pt.y
	haveCursor = true
	left = applyKey(left, l, LeftDown, LeftUp, pt.x, pt.y, tick)
	right = applyKey(right, r, RightDown, RightUp, pt.x, pt.y, tick)
	middle = applyKey(middle, m, MiddleDown, MiddleUp, pt.x, pt.y, tick)
	return left, right, middle
}

type keySample struct {
	down    bool
	pressed bool
}

func readKey(vk uintptr) keySample {
	r, _, _ := procGetAsyncKeyState.Call(vk)
	v := uint16(r)
	return keySample{down: v&0x8000 != 0, pressed: v&0x0001 != 0}
}

func applyKey(was bool, k keySample, downKind, upKind Kind, x, y int32, tick uint32) bool {
	switch {
	case k.down && !was:
		appendSample(downKind, x, y, 0, tick)
	case !k.down && was:
		appendSample(upKind, x, y, 0, tick)
	case !k.down && !was && k.pressed:
		appendSample(downKind, x, y, 0, tick)
		appendSample(upKind, x, y, 0, tick)
	}
	return k.down
}

// notePointer is the test entry for pointer samples. flags uses the same
// RI_MOUSE_* bits as a raw packet: down/up edges and wheel.
func notePointer(x, y int32, flags uint16, wheel int16) {
	if !recOn {
		return
	}
	curX, curY = x, y
	haveCursor = true
	tick := uint32(time.Since(recStart) / time.Millisecond)
	if flags&0x0001 != 0 {
		appendSample(LeftDown, x, y, 0, tick)
	}
	if flags&0x0002 != 0 {
		appendSample(LeftUp, x, y, 0, tick)
	}
	if flags&0x0004 != 0 {
		appendSample(RightDown, x, y, 0, tick)
	}
	if flags&0x0008 != 0 {
		appendSample(RightUp, x, y, 0, tick)
	}
	if flags&0x0010 != 0 {
		appendSample(MiddleDown, x, y, 0, tick)
	}
	if flags&0x0020 != 0 {
		appendSample(MiddleUp, x, y, 0, tick)
	}
	if flags&0x0400 != 0 {
		appendSample(Wheel, x, y, int32(wheel), tick)
	}
}

func appendSample(kind Kind, x, y, wheel int32, tick uint32) {
	if !recOn {
		return
	}
	if kind != Move && pointInRecordedWindow(x, y) {
		return
	}
	if rawN >= maxRecordEvents || time.Since(recStart) >= maxRecordDuration {
		recOn = false
		return
	}
	rawSlots[rawN] = RawEvent{Tick: tick, Kind: kind, X: x, Y: y, Wheel: wheel}
	rawN++
}

func pointInRecordedWindow(x, y int32) bool {
	return x >= winL && x < winR && y >= winT && y < winB
}
