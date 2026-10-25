//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wsCaption        = 0x00C00000
	wsSysMenu        = 0x00080000
	wsMinimizeBox    = 0x00020000
	wsClipChildren   = 0x02000000
	wsChild          = 0x40000000
	wsVisible        = 0x10000000
	wsTabStop        = 0x00010000
	wsExTopmost      = 0x00000008
	wsExClientEdge   = 0x00000200
	esAutoHScroll    = 0x0080
	ssNoPrefix       = 0x0080
	ssLeftNoWordWrap = 0x000C
	bsDefPushButton  = 0x00000001

	windowStyle   = wsCaption | wsSysMenu | wsMinimizeBox | wsClipChildren
	windowExStyle = wsExTopmost

	wmCreate     = 0x0001
	wmDestroy    = 0x0002
	wmClose      = 0x0010
	wmSetFont    = 0x0030
	wmCommand    = 0x0111
	wmTimer      = 0x0113
	wmHotkey     = 0x0312
	wmDpiChanged = 0x02E0
	wmUser       = 0x0400

	wmAppPlaybackDone = wmUser + 1
	emSetLimitText    = 0x00C5

	idCountdown = 101
	idRecord    = 102
	idClear     = 103
	idStart     = 104
	idAbort     = 105

	hotRecord = 1
	hotStart  = 2
	hotAbort  = 3

	timerRecord    = 1
	timerCountdown = 2

	modAlt      = 0x0001
	modShift    = 0x0004
	modNoRepeat = 0x4000
	vk0         = 0x30
	vk9         = 0x39
	vkOEMMinus  = 0xBD

	idcArrow         = 32512
	idiApplication   = 32512
	imageIcon        = 1
	lrShared         = 0x8000
	wmSetIcon        = 0x0080
	iconSmall        = 0
	iconBig          = 1
	smCxIcon         = 11
	smCyIcon         = 12
	smCxSmIcon       = 49
	smCySmIcon       = 50
	colorBtnFace     = 16 // COLOR_BTNFACE + 1
	logPixelsY       = 90
	swHide           = 0
	swShow           = 5
	hwndTopmost      = ^uintptr(0)
	swpNoZOrder      = 0x0004
	swpNoMove        = 0x0002
	swpNoSize        = 0x0001
	swpShowWindow    = 0x0040
	defaultCharset   = 1
	clearTypeQuality = 5
	fwNormal         = 400

	baseClientW = 378
	baseClientH = 70

	hotkeyHint = "Alt+Shift+_ record    ( start    ) stop"
)

type appState int

const (
	stateIdle appState = iota
	stateRecording
	stateCountdown
	statePlaying
)

type application struct {
	hwnd      uintptr
	font      uintptr
	dpi       uint32
	style     uint32
	exStyle   uint32
	state     appState
	events    []Event
	inDPI     bool
	createErr error

	countdownEdit uintptr
	recordBtn     uintptr
	clearBtn      uintptr
	startBtn      uintptr
	abortBtn      uintptr
	statusLabel   uintptr
	hotkeyLabel   uintptr
	note          string

	rec               recorder
	countdownDeadline time.Time
	hotkeyIDs         []uintptr

	playMu    sync.Mutex
	playAbort chan struct{}
}

var (
	clickApp        *application
	wndProcCallback uintptr
	classNameUTF16  = utf16Ptr("ClickWindow")
	appIconBig      uintptr
	appIconSmall    uintptr
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

func runApp() error {
	// The window procedure and GetMessage must stay on the thread that
	// created the window. Without this, the scheduler can move the goroutine
	// and the window stops processing input.
	runtime.LockOSThread()
	enableDPI()
	if wndProcCallback == 0 {
		wndProcCallback = syscall.NewCallback(wndProc)
	}
	if err := registerClass(); err != nil {
		return err
	}
	app := &application{style: windowStyle, exStyle: windowExStyle}
	clickApp = app
	if err := app.createMainWindow(); err != nil {
		return err
	}
	app.registerHotkeys()
	app.refresh()
	app.show()
	procSetFocus.Call(app.countdownEdit)
	// Select the default countdown so a new number replaces it.
	procSendMessageW.Call(app.countdownEdit, 0x00B1, 0, ^uintptr(0))
	app.messageLoop()
	return nil
}

func enableDPI() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is (HANDLE)-4.
	if procSetProcessDpiAwarenessContext.Find() == nil {
		r, _, _ := procSetProcessDpiAwarenessContext.Call(^uintptr(3))
		if r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
}

func registerClass() error {
	inst, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	appIconBig = loadAppIcon(inst, metric(smCxIcon), metric(smCyIcon))
	appIconSmall = loadAppIcon(inst, metric(smCxSmIcon), metric(smCySmIcon))
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		style:         0x0003, // CS_HREDRAW | CS_VREDRAW
		lpfnWndProc:   wndProcCallback,
		hInstance:     inst,
		hIcon:         appIconBig,
		hCursor:       cursor,
		hbrBackground: colorBtnFace,
		lpszClassName: classNameUTF16,
		hIconSm:       appIconSmall,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)
	if atom == 0 {
		if err != nil {
			return fmt.Errorf("register window class: %w", err)
		}
		return errors.New("register window class failed")
	}
	return nil
}

func (a *application) createMainWindow() error {
	a.dpi = screenDPI()
	w, h := a.outerSize()
	x, y := center(w, h)
	title := utf16Ptr("Click")
	hwnd, _, err := procCreateWindowExW.Call(
		uintptr(a.exStyle),
		uintptr(unsafe.Pointer(classNameUTF16)),
		uintptr(unsafe.Pointer(title)),
		uintptr(a.style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0, 0, 0, 0,
	)
	runtime.KeepAlive(title)
	if hwnd == 0 {
		if a.createErr != nil {
			return a.createErr
		}
		if err != nil {
			return fmt.Errorf("create window: %w", err)
		}
		return errors.New("create window failed")
	}
	a.hwnd = hwnd
	if appIconBig != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconBig, appIconBig)
	}
	if appIconSmall != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, appIconSmall)
	}
	return nil
}

func loadAppIcon(inst uintptr, cx, cy int32) uintptr {
	name := utf16Ptr("MAINICON")
	h, _, _ := procLoadImageW.Call(inst, uintptr(unsafe.Pointer(name)), imageIcon, uintptr(cx), uintptr(cy), lrShared)
	runtime.KeepAlive(name)
	if h != 0 {
		return h
	}
	h, _, _ = procLoadIconW.Call(0, idiApplication)
	return h
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	a := clickApp
	if a == nil {
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
	return a.proc(hwnd, msg, wParam, lParam)
}

func (a *application) proc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCreate:
		if err := a.onCreate(hwnd); err != nil {
			a.createErr = err
			if a.font != 0 {
				procDeleteObject.Call(a.font)
				a.font = 0
			}
			return ^uintptr(0)
		}
		return 0
	case wmCommand:
		if hiWord(wParam) == 0 {
			a.onCommand(loWord(wParam))
		}
		return 0
	case wmTimer:
		a.onTimer(wParam)
		return 0
	case wmHotkey:
		a.onHotkey(wParam)
		return 0
	case wmAppPlaybackDone:
		a.onPlaybackDone(wParam != 0)
		return 0
	case wmDpiChanged:
		a.onDPIChanged(wParam, lParam)
		return 0
	case wmDestroy:
		a.shutdown()
		procPostQuitMessage.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

func (a *application) onCreate(hwnd uintptr) error {
	a.hwnd = hwnd
	if dpi := dpiFor(hwnd); dpi != 0 {
		a.dpi = dpi
	}
	a.font = createUIFont(a.dpi)
	if err := a.createControls(); err != nil {
		return err
	}
	a.applyFont()
	a.layoutControls()
	a.resizeToClient(true)
	procSendMessageW.Call(a.countdownEdit, emSetLimitText, 32, 0)
	return nil
}

func (a *application) createControls() error {
	var err error
	child := uint32(wsChild | wsVisible)
	tab := child | wsTabStop
	label := child | ssNoPrefix | ssLeftNoWordWrap
	a.countdownEdit, err = a.ctrlEx("EDIT", "5s", tab|esAutoHScroll, wsExClientEdge, idCountdown)
	if err != nil {
		return err
	}
	a.recordBtn, err = a.ctrl("BUTTON", "Record", tab, idRecord, 0)
	if err != nil {
		return err
	}
	a.clearBtn, err = a.ctrl("BUTTON", "Clear", tab, idClear, 0)
	if err != nil {
		return err
	}
	a.startBtn, err = a.ctrl("BUTTON", "Start", tab|bsDefPushButton, idStart, 0)
	if err != nil {
		return err
	}
	a.abortBtn, err = a.ctrl("BUTTON", "Abort", tab, idAbort, 0)
	if err != nil {
		return err
	}
	a.statusLabel, err = a.ctrl("STATIC", "Ready", label, 0, 0)
	if err != nil {
		return err
	}
	a.hotkeyLabel, err = a.ctrl("STATIC", hotkeyHint, label, 0, 0)
	if err != nil {
		return err
	}
	return nil
}

func (a *application) ctrl(class, text string, style uint32, id int, ex uint32) (uintptr, error) {
	return a.ctrlEx(class, text, style, ex, id)
}

func (a *application) ctrlEx(class, text string, style, ex uint32, id int) (uintptr, error) {
	cls := utf16Ptr(class)
	txt := utf16Ptr(text)
	hwnd, _, err := procCreateWindowExW.Call(
		uintptr(ex),
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(txt)),
		uintptr(style),
		0, 0, 10, 10,
		a.hwnd,
		uintptr(id),
		0, 0,
	)
	runtime.KeepAlive(cls)
	runtime.KeepAlive(txt)
	if hwnd == 0 {
		if err != nil {
			return 0, fmt.Errorf("create %s: %w", class, err)
		}
		return 0, fmt.Errorf("create %s failed", class)
	}
	return hwnd, nil
}

func (a *application) controls() []uintptr {
	return []uintptr{
		a.countdownEdit, a.recordBtn, a.clearBtn, a.startBtn, a.abortBtn,
		a.statusLabel, a.hotkeyLabel,
	}
}

func (a *application) applyFont() {
	for _, h := range a.controls() {
		procSendMessageW.Call(h, wmSetFont, a.font, 1)
	}
}

func (a *application) replaceFont() {
	next := createUIFont(a.dpi)
	if next == 0 {
		return
	}
	old := a.font
	a.font = next
	a.applyFont()
	if old != 0 {
		procDeleteObject.Call(old)
	}
}

func (a *application) layoutControls() {
	a.move(a.countdownEdit, 6, 6, 118, 24)
	a.move(a.recordBtn, 128, 6, 70, 24)
	a.move(a.clearBtn, 202, 6, 54, 24)
	a.move(a.startBtn, 260, 6, 54, 24)
	a.move(a.abortBtn, 318, 6, 54, 24)
	a.move(a.statusLabel, 8, 34, 364, 16)
	a.move(a.hotkeyLabel, 8, 50, 364, 16)
}

func (a *application) move(hwnd uintptr, x, y, w, h int) {
	if hwnd == 0 {
		return
	}
	procMoveWindow.Call(hwnd, uintptr(a.s(x)), uintptr(a.s(y)), uintptr(a.s(w)), uintptr(a.s(h)), 1)
}

func (a *application) s(v int) int32 {
	dpi := a.dpi
	if dpi == 0 {
		dpi = 96
	}
	return int32((int(dpi)*v + 48) / 96)
}

func (a *application) outerSize() (int32, int32) {
	rc := rect{right: a.s(baseClientW), bottom: a.s(baseClientH)}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&rc)), uintptr(a.style), 0, uintptr(a.exStyle))
	return rc.right - rc.left, rc.bottom - rc.top
}

func (a *application) resizeToClient(recenter bool) {
	w, h := a.outerSize()
	x, y := int32(0), int32(0)
	flags := uintptr(swpNoZOrder)
	if recenter {
		x, y = center(w, h)
	} else {
		flags |= swpNoMove
	}
	procSetWindowPos.Call(a.hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), flags)
}

func (a *application) onDPIChanged(wParam, lParam uintptr) {
	if a.inDPI {
		return
	}
	dpi := uint32(wParam >> 16)
	if dpi == 0 {
		return
	}
	a.inDPI = true
	defer func() { a.inDPI = false }()
	a.dpi = dpi
	a.replaceFont()
	a.layoutControls()
	w, h := a.outerSize()
	var x, y int32
	if lParam != 0 {
		r := (*rect)(unsafe.Pointer(lParam))
		x, y = r.left, r.top
	}
	procSetWindowPos.Call(a.hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder)
}

func center(w, h int32) (int32, int32) {
	sx, _, _ := procGetSystemMetrics.Call(0)
	sy, _, _ := procGetSystemMetrics.Call(1)
	x := (int32(sx) - w) / 2
	y := (int32(sy) - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func (a *application) registerHotkeys() {
	const mods = modAlt | modShift | modNoRepeat
	a.tryHotkey(hotRecord, mods, vkOEMMinus)
	a.tryHotkey(hotStart, mods, vk9)
	a.tryHotkey(hotAbort, mods, vk0)
}

func (a *application) tryHotkey(id, mods, vk uintptr) {
	r, _, _ := procRegisterHotKey.Call(a.hwnd, id, mods, vk)
	if r != 0 {
		a.hotkeyIDs = append(a.hotkeyIDs, id)
	}
}

func (a *application) messageLoop() {
	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(r) {
		case 0, -1:
			return
		}
		dlg, _, _ := procIsDialogMessageW.Call(a.hwnd, uintptr(unsafe.Pointer(&m)))
		if dlg != 0 {
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (a *application) onCommand(id uint16) {
	switch id {
	case idRecord:
		a.toggleRecord()
	case idClear:
		a.clearMacro()
	case idStart:
		a.startCountdown()
	case idAbort:
		a.abort()
	}
}

func (a *application) onHotkey(id uintptr) {
	switch id {
	case hotRecord:
		a.toggleRecord()
	case hotStart:
		a.startCountdown()
	case hotAbort:
		a.abort()
	}
}

func (a *application) onTimer(id uintptr) {
	switch id {
	case timerRecord:
		if a.state != stateRecording {
			return
		}
		updateRecordedWindow(a.hwnd)
		n, _, elapsed, on := a.rec.snapshot()
		if !on || overRecordLimit(elapsed, n) {
			a.stopRecord(true)
			return
		}
		a.refresh()
	case timerCountdown:
		a.onCountdownTick()
	}
}

func (a *application) toggleRecord() {
	if a.state == stateRecording {
		a.stopRecord(false)
		return
	}
	if a.state != stateIdle {
		return
	}
	if err := a.rec.start(a.hwnd); err != nil {
		messageBox(a.hwnd, "Click", err.Error())
		return
	}
	a.note = ""
	a.state = stateRecording
	procSetTimer.Call(a.hwnd, timerRecord, 200, 0)
	a.refresh()
}

func (a *application) stopRecord(limited bool) {
	if a.state != stateRecording {
		return
	}
	raw := a.rec.stop()
	procKillTimer.Call(a.hwnd, timerRecord)
	a.events = Build(raw)
	a.state = stateIdle
	switch {
	case limited:
		a.note = "Stopped at limit"
	case len(a.events) == 0:
		a.note = "Nothing recorded"
	default:
		a.note = ""
	}
	a.refresh()
}

func (a *application) clearMacro() {
	if a.state != stateIdle {
		return
	}
	a.events = nil
	a.note = ""
	a.refresh()
}

func (a *application) startCountdown() {
	if a.state != stateIdle {
		return
	}
	if len(a.events) == 0 {
		a.note = "Record a macro first"
		a.refresh()
		return
	}
	delay, err := a.readDelay()
	if err != nil {
		messageBox(a.hwnd, "Click", err.Error())
		return
	}
	if delay == 0 {
		a.beginPlayback()
		return
	}
	a.note = ""
	a.state = stateCountdown
	a.countdownDeadline = time.Now().Add(delay)
	procSetTimer.Call(a.hwnd, timerCountdown, 100, 0)
	a.refresh()
	a.onCountdownTick()
}

func (a *application) onCountdownTick() {
	if a.state != stateCountdown {
		return
	}
	remain := time.Until(a.countdownDeadline)
	if remain <= 0 {
		procKillTimer.Call(a.hwnd, timerCountdown)
		a.beginPlayback()
		return
	}
	a.refresh()
}

func (a *application) beginPlayback() {
	procKillTimer.Call(a.hwnd, timerCountdown)
	events := append([]Event(nil), a.events...)
	if len(events) == 0 {
		a.state = stateIdle
		a.note = "Record a macro first"
		a.refresh()
		return
	}
	abort := make(chan struct{})
	a.playMu.Lock()
	a.playAbort = abort
	a.playMu.Unlock()
	a.note = ""
	a.state = statePlaying
	a.refresh()
	procShowWindow.Call(a.hwnd, swHide)
	go func() {
		// Let the window underneath finish becoming foreground. A click sent
		// in the same instant the window hides is dropped.
		select {
		case <-abort:
			postMessage(a.hwnd, wmAppPlaybackDone, 1, 0)
			return
		case <-time.After(playbackSettle):
		}
		err := playEvents(events, abort)
		wp := uintptr(0)
		if errors.Is(err, errAborted) {
			wp = 1
		}
		postMessage(a.hwnd, wmAppPlaybackDone, wp, 0)
	}()
}

func (a *application) onPlaybackDone(aborted bool) {
	a.playMu.Lock()
	a.playAbort = nil
	a.playMu.Unlock()
	a.show()
	if a.state != statePlaying {
		return
	}
	a.state = stateIdle
	if aborted {
		a.note = "Aborted"
	} else {
		a.note = "Playback finished"
	}
	a.refresh()
}

func (a *application) abort() {
	switch a.state {
	case stateCountdown:
		procKillTimer.Call(a.hwnd, timerCountdown)
		a.state = stateIdle
		a.note = "Aborted"
		a.refresh()
	case statePlaying:
		a.requestAbort()
		a.note = "Aborting"
		a.refresh()
	}
}

func (a *application) requestAbort() {
	a.playMu.Lock()
	defer a.playMu.Unlock()
	if a.playAbort == nil {
		return
	}
	select {
	case <-a.playAbort:
	default:
		close(a.playAbort)
	}
}

func (a *application) readDelay() (time.Duration, error) {
	return parseDelay(getText(a.countdownEdit))
}

func (a *application) refresh() {
	recordLabel := "Record"
	if a.state == stateRecording {
		recordLabel = "Stop"
	}
	setText(a.recordBtn, recordLabel)
	setText(a.statusLabel, a.statusText())

	idle := a.state == stateIdle
	enableWindow(a.recordBtn, idle || a.state == stateRecording)
	enableWindow(a.clearBtn, idle && len(a.events) > 0)
	enableWindow(a.startBtn, idle && len(a.events) > 0)
	enableWindow(a.abortBtn, a.state == stateCountdown || a.state == statePlaying)
	enableWindow(a.countdownEdit, idle)
}

func (a *application) statusText() string {
	switch a.state {
	case stateRecording:
		n, _, elapsed, _ := a.rec.snapshot()
		return fmt.Sprintf("Recording · %d · %s", n, formatLength(elapsed))
	case stateCountdown:
		remain := time.Until(a.countdownDeadline)
		if remain < 0 {
			remain = 0
		}
		return formatRemain(remain)
	case statePlaying:
		if a.note != "" {
			return a.note
		}
		return "Playing"
	default:
		return a.idleStatus()
	}
}

func (a *application) idleStatus() string {
	detail := ""
	if n := len(a.events); n > 0 {
		detail = fmt.Sprintf("%d events · %s", n, formatLength(a.events[n-1].Offset))
	}
	if a.note == "" {
		if detail == "" {
			return "Ready"
		}
		return detail
	}
	if detail == "" {
		return a.note
	}
	return a.note + " · " + detail
}

func (a *application) show() {
	procShowWindow.Call(a.hwnd, swShow)
	procSetWindowPos.Call(a.hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	procSetForegroundWindow.Call(a.hwnd)
}

func (a *application) shutdown() {
	procKillTimer.Call(a.hwnd, timerRecord)
	procKillTimer.Call(a.hwnd, timerCountdown)
	a.requestAbort()
	a.rec.stop()
	for _, id := range a.hotkeyIDs {
		procUnregisterHotKey.Call(a.hwnd, id)
	}
	a.hotkeyIDs = nil
	if a.font != 0 {
		procDeleteObject.Call(a.font)
		a.font = 0
	}
}

func formatLength(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatRemain(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	default:
		return fmt.Sprintf("%dm%ds", m, s)
	}
}

func createUIFont(dpi uint32) uintptr {
	if dpi == 0 {
		dpi = 96
	}
	height := -int32(dpi) * 9 / 72
	face := utf16Ptr("Segoe UI")
	h, _, _ := procCreateFontW.Call(
		uintptr(height),
		0, 0, 0,
		fwNormal,
		0, 0, 0,
		defaultCharset,
		0, 0,
		clearTypeQuality,
		0,
		uintptr(unsafe.Pointer(face)),
	)
	runtime.KeepAlive(face)
	return h
}

func screenDPI() uint32 {
	hdc, _, _ := procGetDC.Call(0)
	dpi, _, _ := procGetDeviceCaps.Call(hdc, logPixelsY)
	procReleaseDC.Call(0, hdc)
	if dpi == 0 {
		return 96
	}
	return uint32(dpi)
}

func dpiFor(hwnd uintptr) uint32 {
	if procGetDpiForWindow.Find() == nil && hwnd != 0 {
		dpi, _, _ := procGetDpiForWindow.Call(hwnd)
		if dpi != 0 {
			return uint32(dpi)
		}
	}
	return screenDPI()
}

func loWord(v uintptr) uint16 { return uint16(v) }
func hiWord(v uintptr) uint16 { return uint16(v >> 16) }
