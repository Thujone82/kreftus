//go:build windows

package main

import (
	"errors"
	"runtime"
	"time"
	"unsafe"
)

const (
	inputMouse = 0

	mouseEventMove        = 0x0001
	mouseEventLeftDown    = 0x0002
	mouseEventLeftUp      = 0x0004
	mouseEventRightDown   = 0x0008
	mouseEventRightUp     = 0x0010
	mouseEventMiddleDown  = 0x0020
	mouseEventMiddleUp    = 0x0040
	mouseEventWheel       = 0x0800
	mouseEventVirtualDesk = 0x4000
	mouseEventAbsolute    = 0x8000

	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79

	// playbackSettle gives the target window time to become foreground after
	// Click hides. minClickHold keeps a tap long enough for the target to see it.
	playbackSettle = 100 * time.Millisecond
	minClickHold   = 40 * time.Millisecond
)

// mouseInput matches Win32 MOUSEINPUT. The extra pointer field forces the
// alignment Windows uses: 28 bytes on 386 and 32 bytes on amd64.
type mouseInput struct {
	dx        int32
	dy        int32
	mouseData uint32
	flags     uint32
	time      uint32
	extra     uintptr
}

// mouseInputPacket matches Win32 INPUT for a mouse event. Go inserts the
// 4-byte pad after typ on amd64, so the struct is 40 bytes there and 28 on 386.
type mouseInputPacket struct {
	typ uint32
	mi  mouseInput
}

var errAborted = errors.New("aborted")

func playEvents(events []Event, abort <-chan struct{}) error {
	if len(events) == 0 {
		return nil
	}
	procTimeBeginPeriod.Call(1)
	defer procTimeEndPeriod.Call(1)

	fromX, fromY, _ := cursorPos()
	start := time.Now()
	prev := time.Duration(0)
	held := Kind(-1)
	var heldAt time.Time
	for _, ev := range events {
		if !glide(fromX, fromY, ev.X, ev.Y, start.Add(prev), start.Add(ev.Offset), abort) {
			return errAborted
		}
		if held >= 0 && isButtonUp(ev.Kind, held) {
			if d := minClickHold - time.Since(heldAt); d > 0 {
				if !sleepAbort(d, abort) {
					return errAborted
				}
			}
			held = -1
		}
		if ev.Kind != Move {
			playOne(ev)
		}
		fromX, fromY = ev.X, ev.Y
		prev = ev.Offset
	}
	return nil
}

func cursorPos() (x, y int32, ok bool) {
	var pt point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, false
	}
	return pt.x, pt.y, true
}

// glide moves in a straight line from the previous point to the next click,
// arriving when the recorded travel time elapses.
func glide(fromX, fromY, toX, toY int32, begin, end time.Time, abort <-chan struct{}) bool {
	if fromX == toX && fromY == toY {
		return waitUntil(end, abort)
	}
	span := end.Sub(begin)
	if span <= 0 {
		sendAbsolute(toX, toY, 0, 0)
		return true
	}
	for {
		if aborted(abort) {
			return false
		}
		elapsed := time.Since(begin)
		if elapsed >= span {
			sendAbsolute(toX, toY, 0, 0)
			return true
		}
		sendAbsolute(lerpCoord(fromX, toX, elapsed, span), lerpCoord(fromY, toY, elapsed, span), 0, 0)
		step := 8 * time.Millisecond
		if remain := span - elapsed; remain < step {
			step = remain
		}
		if !sleepAbort(step, abort) {
			return false
		}
	}
}

func isButtonDown(k Kind) bool {
	return k == LeftDown || k == RightDown || k == MiddleDown
}

func isButtonUp(k, down Kind) bool {
	switch down {
	case LeftDown:
		return k == LeftUp
	case RightDown:
		return k == RightUp
	case MiddleDown:
		return k == MiddleUp
	default:
		return false
	}
}

func playOne(ev Event) {
	button := uint32(0)
	data := int32(0)
	switch ev.Kind {
	case Move:
	case LeftDown:
		button = mouseEventLeftDown
	case LeftUp:
		button = mouseEventLeftUp
	case RightDown:
		button = mouseEventRightDown
	case RightUp:
		button = mouseEventRightUp
	case MiddleDown:
		button = mouseEventMiddleDown
	case MiddleUp:
		button = mouseEventMiddleUp
	case Wheel:
		button = mouseEventWheel
		data = ev.Wheel
	default:
		return
	}
	if !sendAbsolute(ev.X, ev.Y, button, data) {
		procSetCursorPos.Call(uintptr(ev.X), uintptr(ev.Y))
	}
}

// sendAbsolute moves to a screen pixel and, when button is set, applies that
// button in the same input stream. Absolute virtual-desktop coordinates make
// the target window see the move, not only the cursor sprite.
func sendAbsolute(x, y int32, button uint32, data int32) bool {
	ax, ay := normalizeScreen(x, y)
	base := uint32(mouseEventMove | mouseEventAbsolute | mouseEventVirtualDesk)
	pkts := [2]mouseInputPacket{
		{typ: inputMouse, mi: mouseInput{dx: ax, dy: ay, flags: base}},
	}
	n := 1
	if button != 0 {
		pkts[1] = mouseInputPacket{
			typ: inputMouse,
			mi:  mouseInput{dx: ax, dy: ay, mouseData: uint32(data), flags: base | button},
		}
		n = 2
	}
	sent, _, _ := procSendInput.Call(uintptr(n), uintptr(unsafe.Pointer(&pkts[0])), unsafe.Sizeof(pkts[0]))
	runtime.KeepAlive(&pkts)
	return sent == uintptr(n)
}

func normalizeScreen(x, y int32) (int32, int32) {
	vx := metric(smXVirtualScreen)
	vy := metric(smYVirtualScreen)
	vw := metric(smCXVirtualScreen)
	vh := metric(smCYVirtualScreen)
	if vw < 2 {
		vw = 2
	}
	if vh < 2 {
		vh = 2
	}
	ax := scaleToAbsolute(int64(x)-int64(vx), int64(vw))
	ay := scaleToAbsolute(int64(y)-int64(vy), int64(vh))
	return ax, ay
}

func scaleToAbsolute(delta, span int64) int32 {
	if delta < 0 {
		delta = 0
	}
	if delta >= span {
		delta = span - 1
	}
	return int32(delta * 65535 / (span - 1))
}

func metric(index uintptr) int32 {
	v, _, _ := procGetSystemMetrics.Call(index)
	return int32(v)
}

func waitUntil(deadline time.Time, abort <-chan struct{}) bool {
	for {
		if aborted(abort) {
			return false
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			return true
		}
		if remain > 2*time.Millisecond {
			timer := time.NewTimer(remain - time.Millisecond)
			select {
			case <-abort:
				timer.Stop()
				return false
			case <-timer.C:
			}
			continue
		}
		time.Sleep(0)
	}
}

func sleepAbort(d time.Duration, abort <-chan struct{}) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-abort:
		return false
	case <-timer.C:
		return true
	}
}

func aborted(abort <-chan struct{}) bool {
	select {
	case <-abort:
		return true
	default:
		return false
	}
}
