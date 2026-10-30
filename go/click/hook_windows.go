//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Recording polls the cursor, mouse buttons, and keyboard from a normal goroutine.
// A low-level hook, and raw input handled inside the window procedure, both
// stall this process: Windows calls back into Go from inside a message-loop
// syscall, and the window is marked "Not Responding".

const (
	vkLButton  = 0x01
	vkRButton  = 0x02
	vkMButton  = 0x04
	vkXButton1 = 0x05
	vkXButton2 = 0x06
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
	recMouse   = true
	recKeys    = true
	recHwnd    uintptr
	recStart   time.Time
	haveCursor bool
	curX, curY int32
	winL       int32
	winT       int32
	winR       int32
	winB       int32
)

func (r *recorder) start(hwnd uintptr) error {
	return r.startWithOptions(hwnd, true, true)
}

func (r *recorder) startWithOptions(hwnd uintptr, captureMouse, captureKeys bool) error {
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
	recMouse = captureMouse
	recKeys = captureKeys
	recHwnd = hwnd
	recMu.Unlock()
	updateRecordedWindow(hwnd)

	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	go pollInput(r.stopCh, r.doneCh)
	return nil
}

func updateRecordedWindow(hwnd uintptr) {
	var rc rect
	if hwnd != 0 {
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis != 0 {
			procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		}
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

// isPolledVK reports whether vk should be polled as a keyboard key.
// Generic Shift/Ctrl/Alt (0x10..0x12) are skipped in favor of their L/R variants (0xA0..0xA5)
// so each modifier press produces a single KeyDown/KeyUp pair.
func isPolledVK(vk uint16) bool {
	if vk < 0x08 || vk > 0xFE {
		return false
	}
	switch vk {
	case vkShift, vkControl, vkMenu:
		return false
	case 0x0A, 0x0B, 0x0E, 0x0F, 0x16, 0x1A, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40:
		return false
	case 0x5E, 0x88, 0x89, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F:
		return false
	case 0x97, 0x98, 0x99, 0x9A, 0x9B, 0x9C, 0x9D, 0x9E, 0x9F:
		return false
	case 0xB8, 0xB9, 0xE0, 0xE1, 0xE3, 0xE4, 0xE6, 0xE8:
		return false
	default:
		return true
	}
}

func pollInput(stop <-chan struct{}, done chan<- struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	procTimeBeginPeriod.Call(1)
	defer procTimeEndPeriod.Call(1)
	// Drop the "pressed since last call" bit from before recording started,
	// and initialize current key states so held start-hotkey keys aren't recorded.
	readKey(vkLButton)
	readKey(vkRButton)
	readKey(vkMButton)
	var keyWas [256]bool
	for vk := uint16(0x08); vk <= 0xFE; vk++ {
		if isPolledVK(vk) {
			ks := readKey(uintptr(vk))
			keyWas[vk] = ks.down
		}
	}
	var left, right, middle bool
	ticker := time.NewTicker(4 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			// Perform one final sample before exiting so an action completed right
			// before stopping is never missed.
			sampleInput(left, right, middle, &keyWas)
			return
		case <-ticker.C:
			left, right, middle = sampleInput(left, right, middle, &keyWas)
		}
	}
}

func sampleInput(left, right, middle bool, keyWas *[256]bool) (bool, bool, bool) {
	var pt point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return left, right, middle
	}
	l := readKey(vkLButton)
	r := readKey(vkRButton)
	m := readKey(vkMButton)

	fg, _, _ := procGetForegroundWindow.Call()

	recMu.Lock()
	if !recOn {
		recMu.Unlock()
		return left, right, middle
	}
	doMouse := recMouse
	doKeys := recKeys
	selfHwnd := recHwnd
	tick := uint32(time.Since(recStart) / time.Millisecond)
	curX, curY = pt.x, pt.y
	haveCursor = true
	if doMouse {
		left = applyKey(left, l, LeftDown, LeftUp, pt.x, pt.y, tick)
		right = applyKey(right, r, RightDown, RightUp, pt.x, pt.y, tick)
		middle = applyKey(middle, m, MiddleDown, MiddleUp, pt.x, pt.y, tick)
	}
	recMu.Unlock()

	if !doKeys {
		return left, right, middle
	}
	// Ignore keystrokes typed while the Click window itself is in the foreground.
	if selfHwnd != 0 && fg == selfHwnd {
		for vk := uint16(0x08); vk <= 0xFE; vk++ {
			if isPolledVK(vk) {
				ks := readKey(uintptr(vk))
				keyWas[vk] = ks.down
			}
		}
		return left, right, middle
	}

	var kSamples [256]keySample
	for vk := uint16(0x08); vk <= 0xFE; vk++ {
		if isPolledVK(vk) {
			kSamples[vk] = readKey(uintptr(vk))
		}
	}

	recMu.Lock()
	defer recMu.Unlock()
	if !recOn {
		return left, right, middle
	}
	// Process modifiers first so chord state is up to date before non-modifier keys on the same tick.
	for vk := uint16(0x08); vk <= 0xFE; vk++ {
		if isPolledVK(vk) && isModifierVK(vk) {
			keyWas[vk] = applyVK(keyWas[vk], kSamples[vk], vk, tick)
		}
	}
	for vk := uint16(0x08); vk <= 0xFE; vk++ {
		if isPolledVK(vk) && !isModifierVK(vk) {
			keyWas[vk] = applyVK(keyWas[vk], kSamples[vk], vk, tick)
		}
	}
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
		if !appendSample(downKind, x, y, 0, tick) {
			return false
		}
	case !k.down && was:
		appendSample(upKind, x, y, 0, tick)
	case !k.down && !was && k.pressed:
		if appendSample(downKind, x, y, 0, tick) {
			appendSample(upKind, x, y, 0, tick)
		}
	}
	return k.down
}

func applyVK(was bool, k keySample, vk uint16, tick uint32) bool {
	switch {
	case k.down && !was:
		appendKeySample(KeyDown, vk, tick)
	case !k.down && was:
		appendKeySample(KeyUp, vk, tick)
	case !k.down && !was && k.pressed:
		appendKeySample(KeyDown, vk, tick)
		appendKeySample(KeyUp, vk, tick)
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

func appendSample(kind Kind, x, y, wheel int32, tick uint32) bool {
	if !recOn {
		return false
	}
	// Never discard a button-up event if its button-down started outside the Click window;
	// otherwise moving the cursor onto the Click window to press Stop would eat the final action.
	if kind != Move && kind != LeftUp && kind != RightUp && kind != MiddleUp && pointInRecordedWindow(x, y) {
		return false
	}
	if rawN >= maxRecordEvents || time.Since(recStart) >= maxRecordDuration {
		recOn = false
		return false
	}
	rawSlots[rawN] = RawEvent{Tick: tick, Kind: kind, X: x, Y: y, Wheel: wheel}
	rawN++
	return true
}

func appendKeySample(kind Kind, vk uint16, tick uint32) {
	if !recOn {
		return
	}
	if rawN >= maxRecordEvents || time.Since(recStart) >= maxRecordDuration {
		recOn = false
		return
	}
	rawSlots[rawN] = RawEvent{Tick: tick, Kind: kind, VK: vk}
	rawN++
}

func pointInRecordedWindow(x, y int32) bool {
	return x >= winL && x < winR && y >= winT && y < winB
}

// PickNextClick waits until the user presses the left mouse button anywhere on screen
// (or abort is closed / timeout expires) and returns the clicked physical screen pixel.
func PickNextClick(abort <-chan struct{}, timeout time.Duration) (int32, int32, bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	readKey(vkLButton)
	readKey(vkEscape)
	deadline := time.Now().Add(timeout)
	wasDown := readKey(vkLButton).down
	ticker := time.NewTicker(8 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		select {
		case <-abort:
			return 0, 0, false
		case <-ticker.C:
			if readKey(vkEscape).down {
				return 0, 0, false
			}
			ks := readKey(vkLButton)
			if (ks.down && !wasDown) || (!ks.down && !wasDown && ks.pressed) {
				x, y, ok := cursorPos()
				return x, y, ok
			}
			wasDown = ks.down
		}
	}
	return 0, 0, false
}
