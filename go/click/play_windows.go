//go:build windows

package main

import (
	"errors"
	"math"
	"runtime"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	inputMouse    = 0
	inputKeyboard = 1

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

	keyEventExtendedKey = 0x0001
	keyEventKeyUp       = 0x0002
	keyEventUnicode     = 0x0004
	keyEventScancode    = 0x0008

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

// keybdInput matches Win32 KEYBDINPUT, padded to the size of MOUSEINPUT
// (28 bytes on 386, 32 bytes on amd64) so keybdInputPacket has the exact
// same size and alignment as mouseInputPacket on both architectures.
type keybdInput struct {
	wVk     uint16
	wScan   uint16
	dwFlags uint32
	time    uint32
	dwExtra uintptr
	pad     [8]byte
}

type keybdInputPacket struct {
	typ uint32
	ki  keybdInput
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
			if isButtonDown(ev.Kind) {
				held = ev.Kind
				heldAt = time.Now()
			}
		}
		fromX, fromY = ev.X, ev.Y
		prev = ev.Offset
	}
	return nil
}

// playSteps executes a slice of high-level Steps with optional speed scaling
// and step-index progress reporting. On abort or return, any buttons or modifiers
// pressed during playback are guaranteed to be released.
func playSteps(steps []Step, speed float64, abort <-chan struct{}, onStep func(idx int)) error {
	if len(steps) == 0 {
		return nil
	}
	if speed <= 0 {
		speed = 1.0
	}
	procTimeBeginPeriod.Call(1)
	defer procTimeEndPeriod.Call(1)

	scaleDur := func(ms int64) time.Duration {
		if ms <= 0 {
			return 0
		}
		scaled := float64(ms) / speed
		return time.Duration(math.Round(scaled)) * time.Millisecond
	}

	var (
		heldBtns = make(map[Button]bool)
		heldVKs  = make(map[uint16]bool)
	)
	defer func() {
		// Safety release: ensure no mouse button or modifier key remains stuck down.
		cx, cy, _ := cursorPos()
		for btn, down := range heldBtns {
			if down {
				_, upFlag := buttonFlags(btn)
				sendAbsolute(cx, cy, upFlag, 0)
			}
		}
		for vk, down := range heldVKs {
			if down {
				sendKeyVK(vk, true)
			}
		}
	}()

	fromX, fromY, _ := cursorPos()

	for idx, st := range steps {
		if aborted(abort) {
			return errAborted
		}
		if onStep != nil {
			onStep(idx)
		}
		delay := scaleDur(st.EffectiveDelayMs(idx))
		hold := scaleDur(st.HoldMs)
		if hold < minClickHold {
			hold = minClickHold
		}

		switch st.Type {
		case StepWait:
			waitDur := delay
			if st.HoldMs > 0 {
				waitDur += scaleDur(st.HoldMs)
			}
			if waitDur > 0 && !sleepAbort(waitDur, abort) {
				return errAborted
			}

		case StepClick, StepDoubleClick:
			if st.Glide {
				now := time.Now()
				if !glide(fromX, fromY, st.X, st.Y, now, now.Add(delay), abort) {
					return errAborted
				}
			} else {
				if delay > 0 && !sleepAbort(delay, abort) {
					return errAborted
				}
				sendAbsolute(st.X, st.Y, 0, 0)
			}
			fromX, fromY = st.X, st.Y
			downFlag, upFlag := buttonFlags(st.Button)
			clicks := 1
			if st.Type == StepDoubleClick {
				clicks = 2
			}
			for c := 0; c < clicks; c++ {
				heldBtns[st.Button] = true
				sendAbsolute(st.X, st.Y, downFlag, 0)
				tapHold := hold
				if clicks == 2 && tapHold > 65*time.Millisecond {
					tapHold = 50 * time.Millisecond
				}
				if !sleepAbort(tapHold, abort) {
					return errAborted
				}
				sendAbsolute(st.X, st.Y, upFlag, 0)
				heldBtns[st.Button] = false
				if c == 0 && clicks == 2 {
					if !sleepAbort(45*time.Millisecond, abort) {
						return errAborted
					}
				}
			}

		case StepDrag:
			if st.Glide {
				now := time.Now()
				if !glide(fromX, fromY, st.X, st.Y, now, now.Add(delay), abort) {
					return errAborted
				}
			} else {
				if delay > 0 && !sleepAbort(delay, abort) {
					return errAborted
				}
				sendAbsolute(st.X, st.Y, 0, 0)
			}
			downFlag, upFlag := buttonFlags(st.Button)
			heldBtns[st.Button] = true
			sendAbsolute(st.X, st.Y, downFlag, 0)
			if !sleepAbort(minClickHold, abort) {
				return errAborted
			}
			dragSpan := hold
			if dragSpan < 80*time.Millisecond {
				dragSpan = 120 * time.Millisecond
			}
			now := time.Now()
			if !glide(st.X, st.Y, st.X2, st.Y2, now, now.Add(dragSpan), abort) {
				return errAborted
			}
			sendAbsolute(st.X2, st.Y2, upFlag, 0)
			heldBtns[st.Button] = false
			fromX, fromY = st.X2, st.Y2

		case StepWheel:
			targetX, targetY := st.X, st.Y
			if targetX == 0 && targetY == 0 {
				targetX, targetY = fromX, fromY
			}
			if st.Glide && (targetX != fromX || targetY != fromY) {
				now := time.Now()
				if !glide(fromX, fromY, targetX, targetY, now, now.Add(delay), abort) {
					return errAborted
				}
			} else if delay > 0 {
				if !sleepAbort(delay, abort) {
					return errAborted
				}
			}
			wheelVal := st.Wheel
			if wheelVal == 0 {
				wheelVal = -120
			}
			sendAbsolute(targetX, targetY, mouseEventWheel, wheelVal)
			fromX, fromY = targetX, targetY

		case StepKey:
			if delay > 0 && !sleepAbort(delay, abort) {
				return errAborted
			}
			vk := st.Key
			if vk == 0 && st.KeyName != "" {
				if resolved, ok := NameToVK(st.KeyName); ok {
					vk = resolved
				}
			}
			modVKs := expandModVKs(st.Mods)
			for _, mvk := range modVKs {
				heldVKs[mvk] = true
				sendKeyVK(mvk, false)
			}
			if vk != 0 {
				heldVKs[vk] = true
				sendKeyVK(vk, false)
				if !sleepAbort(hold, abort) {
					return errAborted
				}
				sendKeyVK(vk, true)
				heldVKs[vk] = false
			}
			for i := len(modVKs) - 1; i >= 0; i-- {
				mvk := modVKs[i]
				sendKeyVK(mvk, true)
				heldVKs[mvk] = false
			}

		case StepText:
			if delay > 0 && !sleepAbort(delay, abort) {
				return errAborted
			}
			u16 := utf16.Encode([]rune(st.Text))
			for _, code := range u16 {
				if aborted(abort) {
					return errAborted
				}
				sendUnicodeChar(code, false)
				sendUnicodeChar(code, true)
				if !sleepAbort(8*time.Millisecond, abort) {
					return errAborted
				}
			}
		}
	}
	return nil
}

func buttonFlags(btn Button) (downFlag, upFlag uint32) {
	switch btn {
	case ButtonRight:
		return mouseEventRightDown, mouseEventRightUp
	case ButtonMiddle:
		return mouseEventMiddleDown, mouseEventMiddleUp
	default:
		return mouseEventLeftDown, mouseEventLeftUp
	}
}

func expandModVKs(mods Mods) []uint16 {
	var out []uint16
	if mods.Has(ModCtrl) {
		out = append(out, vkLControl)
	}
	if mods.Has(ModAlt) {
		out = append(out, vkLMenu)
	}
	if mods.Has(ModShift) {
		out = append(out, vkLShift)
	}
	if mods.Has(ModWin) {
		out = append(out, vkLWin)
	}
	return out
}

func isExtendedVK(vk uint16) bool {
	switch vk {
	case vkPrior, vkNext, vkEnd, vkHome, vkLeft, vkUp, vkRight, vkDown,
		vkInsert, vkDelete, vkDivide, vkNumlock, vkRControl, vkRMenu, vkLWin, vkRWin, vkApps, vkSnapshot:
		return true
	default:
		return false
	}
}

func sendKeyVK(vk uint16, up bool) bool {
	scan, _, _ := procMapVirtualKeyW.Call(uintptr(vk), 0) // MAPVK_VK_TO_VSC
	flags := uint32(0)
	if isExtendedVK(vk) {
		flags |= keyEventExtendedKey
	}
	if up {
		flags |= keyEventKeyUp
	}
	pkt := keybdInputPacket{
		typ: inputKeyboard,
		ki: keybdInput{
			wVk:     vk,
			wScan:   uint16(scan),
			dwFlags: flags,
		},
	}
	sent, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&pkt)), unsafe.Sizeof(pkt))
	runtime.KeepAlive(&pkt)
	return sent == 1
}

func sendUnicodeChar(ch uint16, up bool) bool {
	flags := uint32(keyEventUnicode)
	if up {
		flags |= keyEventKeyUp
	}
	pkt := keybdInputPacket{
		typ: inputKeyboard,
		ki: keybdInput{
			wVk:     0,
			wScan:   ch,
			dwFlags: flags,
		},
	}
	sent, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&pkt)), unsafe.Sizeof(pkt))
	runtime.KeepAlive(&pkt)
	return sent == 1
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
