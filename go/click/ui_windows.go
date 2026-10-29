//go:build windows

package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
)

//go:embed web/index.html web/app.css web/app.js
var webAssets embed.FS

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

	wmCreate          = 0x0001
	wmDestroy         = 0x0002
	wmClose           = 0x0010
	wmTimeChange      = 0x001E
	wmSetFont         = 0x0030
	wmSetIcon         = 0x0080
	wmCommand         = 0x0111
	wmTimer           = 0x0113
	wmPowerBroadcast  = 0x0218
	wmDpiChanged      = 0x02E0
	wmHotkey          = 0x0312
	wmUser            = 0x0400

	wmAppPlaybackDone = wmUser + 1
	wmAppScheduleFire = wmUser + 2

	hotRecord = 1
	hotStart  = 2
	hotAbort  = 3

	timerRecord    = 1
	timerCountdown = 2

	modAlt      = 0x0001
	modCtrl     = 0x0002
	modShift    = 0x0004
	modWin      = 0x0008
	modNoRepeat = 0x4000

	idcArrow       = 32512
	idiApplication = 32512
	imageIcon      = 1
	lrShared       = 0x8000
	iconSmall      = 0
	iconBig        = 1
	smCxIcon       = 11
	smCyIcon       = 12
	smCxSmIcon     = 49
	smCySmIcon     = 50
	colorBtnFace   = 16
	swHide         = 0
	swShow         = 5
	swRestore      = 9
	hwndTopmost    = ^uintptr(0)
	hwndNoTopmost  = ^uintptr(1)
	swpNoZOrder    = 0x0004
	swpNoMove      = 0x0002
	swpNoSize      = 0x0001
	swpShowWindow  = 0x0040
)

type appState int

const (
	stateIdle appState = iota
	stateRecording
	stateCountdown
	statePlaying
)

type queuedRun struct {
	sched Schedule
}

type application struct {
	hwnd    uintptr // main WebView2 host window
	msgHwnd uintptr // hidden Win32 helper window for hotkeys, timers, tray, power broadcasts
	wv      webview2.WebView
	store   *Store
	tray    trayIcon

	state             appState
	note              string
	activeMacroID     string
	appendToMacroID   string
	playFromStep      int
	playingStepIdx    atomic.Int32
	activeSchedule    *Schedule
	countdownDeadline time.Time
	hotkeyIDs         []uintptr

	rec       recorder
	playMu    sync.Mutex
	playAbort chan struct{}

	schedWake chan struct{}
	schedStop chan struct{}
	runQueue  []queuedRun
}

var (
	clickApp        *application
	msgWndCallback  uintptr
	msgClassName    = utf16Ptr("ClickMsgHostWindow")
	appIconBig      uintptr
	appIconSmall    uintptr
	gwlpWndProcIdx  = ^uintptr(3) // -4 (GWLP_WNDPROC)
	origWebViewProc uintptr
	user32SetWndLongPtr = user32.NewProc("SetWindowLongPtrW")
	user32SetWndLong    = user32.NewProc("SetWindowLongW")
	user32CallWndProc   = user32.NewProc("CallWindowProcW")
	wvSubCallback       uintptr
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
	runtime.LockOSThread()
	enableDPI()

	store, err := NewStore("")
	if err != nil {
		return err
	}

	app := &application{
		store:     store,
		schedWake: make(chan struct{}, 1),
		schedStop: make(chan struct{}),
	}
	app.playingStepIdx.Store(-1)
	macros, _, _, _ := store.Snapshot()
	if len(macros) > 0 {
		app.activeMacroID = macros[0].ID
	}
	clickApp = app

	// Keep WebView2 user-data cache inside %APPDATA%\Click\webview2 so the exe directory stays clean.
	wvDataDir := filepath.Join(os.Getenv("APPDATA"), "Click", "webview2")
	_ = os.MkdirAll(wvDataDir, 0o755)

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  wvDataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  "Click",
			Width:  1180,
			Height: 760,
			Center: true,
		},
	})
	if w == nil {
		return errors.New("failed to initialize WebView2 runtime; please ensure Microsoft Edge WebView2 Runtime is installed")
	}
	app.wv = w
	app.hwnd = uintptr(w.Window())
	w.SetSize(960, 620, webview2.HintMin)

	inst, _, _ := procGetModuleHandleW.Call(0)
	appIconBig = loadAppIcon(inst, metric(smCxIcon), metric(smCyIcon))
	appIconSmall = loadAppIcon(inst, metric(smCxSmIcon), metric(smCySmIcon))
	if appIconBig != 0 {
		procSendMessageW.Call(app.hwnd, wmSetIcon, iconBig, appIconBig)
	}
	if appIconSmall != 0 {
		procSendMessageW.Call(app.hwnd, wmSetIcon, iconSmall, appIconSmall)
	}

	// Subclass the WebView2 host window so WM_CLOSE can minimize to the tray when schedules are enabled.
	subclassWebViewWindow(app.hwnd)

	if err := app.createMessageWindow(inst); err != nil {
		return err
	}
	app.tray.init(app.msgHwnd, appIconSmall, "Click — Macro Manager & Scheduler")
	app.rebindHotkeys()
	app.applyKeepAwake()

	if err := app.registerBridge(); err != nil {
		return err
	}

	pageHTML, err := buildEmbeddedHTML()
	if err != nil {
		return err
	}
	w.SetHtml(pageHTML)

	go app.schedulerLoop()
	w.Run()
	app.shutdown()
	return nil
}

func buildEmbeddedHTML() (string, error) {
	htmlBytes, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		return "", err
	}
	cssBytes, err := webAssets.ReadFile("web/app.css")
	if err != nil {
		return "", err
	}
	jsBytes, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		return "", err
	}
	page := string(htmlBytes)
	page = replaceOnce(page, "/*__INLINE_CSS__*/", string(cssBytes))
	page = replaceOnce(page, "//__INLINE_JS__", string(jsBytes))
	return page, nil
}

func replaceOnce(s, old, repl string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + repl + s[i+len(old):]
		}
	}
	return s
}

func subclassWebViewWindow(hwnd uintptr) {
	if wvSubCallback == 0 {
		wvSubCallback = syscall.NewCallback(webViewSubclassProc)
	}
	if unsafe.Sizeof(uintptr(0)) == 8 && user32SetWndLongPtr.Find() == nil {
		origWebViewProc, _, _ = user32SetWndLongPtr.Call(hwnd, gwlpWndProcIdx, wvSubCallback)
	} else {
		origWebViewProc, _, _ = user32SetWndLong.Call(hwnd, gwlpWndProcIdx, wvSubCallback)
	}
}

func webViewSubclassProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	a := clickApp
	if a != nil && msg == wmClose {
		if a.store.HasActiveSchedules() {
			procShowWindow.Call(hwnd, swHide)
			a.tray.showBalloon("Click is still running", "Active schedules will continue running in the background. Right-click the tray icon to exit.")
			return 0
		}
	}
	ret, _, _ := user32CallWndProc.Call(origWebViewProc, hwnd, msg, wParam, lParam)
	return ret
}

func (a *application) createMessageWindow(inst uintptr) error {
	if msgWndCallback == 0 {
		msgWndCallback = syscall.NewCallback(msgWndProc)
	}
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   msgWndCallback,
		hInstance:     inst,
		lpszClassName: msgClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)

	title := utf16Ptr("ClickMsgHost")
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(msgClassName)),
		uintptr(unsafe.Pointer(title)),
		0,
		0, 0, 0, 0,
		0, 0, inst, 0,
	)
	runtime.KeepAlive(title)
	if hwnd == 0 {
		return fmt.Errorf("create message window: %w", err)
	}
	a.msgHwnd = hwnd
	return nil
}

func msgWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	a := clickApp
	if a == nil {
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
	switch msg {
	case wmTimer:
		a.onTimer(wParam)
		return 0
	case wmHotkey:
		a.onHotkey(wParam)
		return 0
	case wmAppPlaybackDone:
		a.onPlaybackDone(wParam != 0)
		return 0
	case wmAppScheduleFire:
		a.onSchedulerCheckDue()
		return 0
	case wmPowerBroadcast, wmTimeChange:
		a.wakeScheduler()
		return 0
	case wmTrayCallback:
		ev := uint32(lParam)
		switch ev {
		case wmLButtonUp, wmLButtonDblClk:
			a.showMainWindow()
		case wmRButtonUp:
			cmd := showTrayPopupMenu(hwnd, a.state == stateRecording)
			switch cmd {
			case trayMenuShow:
				a.showMainWindow()
			case trayMenuRecord:
				a.appendToMacroID = ""
				a.toggleRecord()
			case trayMenuAbort:
				a.abort()
			case trayMenuExit:
				a.wv.Terminate()
			}
		}
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

func enableDPI() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		r, _, _ := procSetProcessDpiAwarenessContext.Call(^uintptr(3))
		if r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
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

func (a *application) rebindHotkeys() {
	if a.msgHwnd == 0 {
		return
	}
	for _, id := range a.hotkeyIDs {
		procUnregisterHotKey.Call(a.msgHwnd, id)
	}
	a.hotkeyIDs = nil

	cfg := a.store.GetSettings()
	a.tryHotkey(hotRecord, uintptr(cfg.HotkeyRecord.Mods)|modNoRepeat, uintptr(cfg.HotkeyRecord.VK))
	a.tryHotkey(hotStart, uintptr(cfg.HotkeyPlay.Mods)|modNoRepeat, uintptr(cfg.HotkeyPlay.VK))
	a.tryHotkey(hotAbort, uintptr(cfg.HotkeyAbort.Mods)|modNoRepeat, uintptr(cfg.HotkeyAbort.VK))
}

func (a *application) tryHotkey(id, mods, vk uintptr) {
	if vk == 0 {
		return
	}
	r, _, _ := procRegisterHotKey.Call(a.msgHwnd, id, mods, vk)
	if r != 0 {
		a.hotkeyIDs = append(a.hotkeyIDs, id)
	}
}

func (a *application) applyKeepAwake() {
	cfg := a.store.GetSettings()
	setKeepAwake(cfg.KeepAwake && a.store.HasActiveSchedules())
}

func (a *application) dispatchUI(fn func()) {
	if a.wv != nil {
		a.wv.Dispatch(fn)
	}
}

func (a *application) pushState() {
	if a.wv == nil {
		return
	}
	snap := a.buildSnapshot()
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	js := "window.__onClickState && window.__onClickState(" + string(data) + ");"
	a.wv.Dispatch(func() {
		a.wv.Eval(js)
	})
}

func (a *application) pushToast(kind, message string) {
	if a.wv == nil {
		return
	}
	bKind, _ := json.Marshal(kind)
	bMsg, _ := json.Marshal(message)
	js := fmt.Sprintf("window.__onClickToast && window.__onClickToast(%s, %s);", string(bKind), string(bMsg))
	a.wv.Dispatch(func() {
		a.wv.Eval(js)
	})
}

func (a *application) onHotkey(id uintptr) {
	switch id {
	case hotRecord:
		a.appendToMacroID = ""
		a.toggleRecord()
	case hotStart:
		cfg := a.store.GetSettings()
		a.startCountdownWith(a.activeMacroID, cfg.DefaultCountdown, 0)
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
		a.pushState()
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
	cfg := a.store.GetSettings()
	if err := a.rec.startWithOptions(a.hwnd, cfg.RecordMouse, cfg.RecordKeyboard); err != nil {
		a.pushToast("error", err.Error())
		return
	}
	a.note = ""
	a.state = stateRecording
	procSetTimer.Call(a.msgHwnd, timerRecord, 200, 0)
	a.pushState()
}

func (a *application) stopRecord(limited bool) {
	if a.state != stateRecording {
		return
	}
	raw := a.rec.stop()
	procKillTimer.Call(a.msgHwnd, timerRecord)
	raw = TrimTrailingHotkey(raw)

	cfg := a.store.GetSettings()
	steps := BuildSteps(raw, DefaultBuildOptions(cfg.MergeTyping))
	a.state = stateIdle

	switch {
	case limited:
		a.note = "Stopped at limit"
	case len(steps) == 0:
		a.note = "Nothing recorded"
	default:
		a.note = fmt.Sprintf("Recorded %d steps", len(steps))
	}

	if len(steps) > 0 {
		if a.appendToMacroID != "" {
			if existing, ok := a.store.GetMacro(a.appendToMacroID); ok {
				existing.Steps = append(existing.Steps, steps...)
				saved, err := a.store.UpsertMacro(existing)
				if err == nil {
					a.activeMacroID = saved.ID
					a.pushToast("success", fmt.Sprintf("Appended %d steps to %s", len(steps), saved.Name))
				}
			}
		} else {
			macros, _, _, _ := a.store.Snapshot()
			name := NextMacroName(macros, time.Now())
			saved, err := a.store.UpsertMacro(Macro{
				Name:  name,
				Steps: steps,
				Speed: 1.0,
			})
			if err == nil {
				a.activeMacroID = saved.ID
				a.pushToast("success", fmt.Sprintf("Saved %s (%d steps)", saved.Name, len(steps)))
			}
		}
	}
	a.appendToMacroID = ""
	a.pushState()
}

func (a *application) startCountdownWith(macroID string, delayStr string, fromStep int) {
	if a.state != stateIdle {
		return
	}
	m, ok := a.store.GetMacro(macroID)
	if !ok || len(m.Steps) == 0 {
		a.note = "Select or record a macro with steps first"
		a.pushToast("warn", a.note)
		a.pushState()
		return
	}
	if delayStr == "" {
		delayStr = a.store.GetSettings().DefaultCountdown
	}
	delay, err := parseDelay(delayStr)
	if err != nil {
		a.pushToast("error", err.Error())
		return
	}
	a.activeMacroID = m.ID
	a.playFromStep = fromStep
	a.activeSchedule = nil
	if delay == 0 {
		a.beginPlayback()
		return
	}
	a.note = ""
	a.state = stateCountdown
	a.countdownDeadline = time.Now().Add(delay)
	procSetTimer.Call(a.msgHwnd, timerCountdown, 100, 0)
	a.pushState()
}

func (a *application) onCountdownTick() {
	if a.state != stateCountdown {
		return
	}
	remain := time.Until(a.countdownDeadline)
	if remain <= 0 {
		procKillTimer.Call(a.msgHwnd, timerCountdown)
		a.beginPlayback()
		return
	}
	a.pushState()
}

func (a *application) beginPlayback() {
	procKillTimer.Call(a.msgHwnd, timerCountdown)
	m, ok := a.store.GetMacro(a.activeMacroID)
	if !ok || len(m.Steps) == 0 {
		a.state = stateIdle
		a.note = "Record a macro first"
		a.pushState()
		return
	}

	startIdx := a.playFromStep
	if startIdx < 0 || startIdx >= len(m.Steps) {
		startIdx = 0
	}
	steps := append([]Step(nil), m.Steps[startIdx:]...)
	speed := m.Speed
	if speed <= 0 {
		speed = 1.0
	}
	plays := 1
	var gap time.Duration
	if a.activeSchedule != nil {
		if a.activeSchedule.PlaysPerRun > 1 {
			plays = a.activeSchedule.PlaysPerRun
		}
		if a.activeSchedule.PlayGapMs > 0 {
			gap = time.Duration(a.activeSchedule.PlayGapMs) * time.Millisecond
		}
	}

	abort := make(chan struct{})
	a.playMu.Lock()
	a.playAbort = abort
	a.playMu.Unlock()

	a.note = ""
	a.state = statePlaying
	a.playingStepIdx.Store(int32(startIdx))
	a.pushState()

	cfg := a.store.GetSettings()
	wasVisible, _, _ := procIsWindowVisible.Call(a.hwnd)
	if cfg.HideWhilePlaying && wasVisible != 0 {
		procShowWindow.Call(a.hwnd, swHide)
	}

	go func() {
		select {
		case <-abort:
			postMessage(a.msgHwnd, wmAppPlaybackDone, 1, 0)
			return
		case <-time.After(playbackSettle):
		}

		var err error
		for p := 0; p < plays; p++ {
			err = playSteps(steps, speed, abort, func(idx int) {
				a.playingStepIdx.Store(int32(startIdx + idx))
				a.pushState()
			})
			if err != nil {
				break
			}
			if p+1 < plays && gap > 0 {
				if !sleepAbort(gap, abort) {
					err = errAborted
					break
				}
			}
		}

		wp := uintptr(0)
		if errors.Is(err, errAborted) {
			wp = 1
		}
		postMessage(a.msgHwnd, wmAppPlaybackDone, wp, uintptr(wasVisible))
	}()
}

func (a *application) onPlaybackDone(aborted bool) {
	a.playMu.Lock()
	a.playAbort = nil
	a.playMu.Unlock()
	a.playingStepIdx.Store(-1)

	cfg := a.store.GetSettings()
	if cfg.HideWhilePlaying && a.activeSchedule == nil {
		a.showMainWindow()
	}

	if a.state != statePlaying {
		return
	}
	a.state = stateIdle

	m, _ := a.store.GetMacro(a.activeMacroID)
	res := "done"
	if aborted {
		a.note = "Aborted"
		res = "aborted"
	} else {
		a.note = "Playback finished"
	}

	trig := "manual"
	schedID := ""
	schedName := ""
	if a.activeSchedule != nil {
		trig = "schedule"
		schedID = a.activeSchedule.ID
		schedName = a.activeSchedule.Name
		a.finalizeScheduleRun(schedID, res)
		a.activeSchedule = nil
	}
	_ = a.store.AppendHistory(HistoryEntry{
		Time:       time.Now(),
		MacroID:    m.ID,
		MacroName:  m.Name,
		ScheduleID: schedID,
		SchedName:  schedName,
		Trigger:    trig,
		Result:     res,
	})

	a.pushState()

	// If another scheduled run was queued while we were playing, start it now.
	if len(a.runQueue) > 0 {
		next := a.runQueue[0]
		a.runQueue = a.runQueue[1:]
		a.triggerScheduledMacro(next.sched)
	}
}

func (a *application) abort() {
	switch a.state {
	case stateCountdown:
		procKillTimer.Call(a.msgHwnd, timerCountdown)
		a.state = stateIdle
		a.note = "Aborted"
		if a.activeSchedule != nil {
			m, _ := a.store.GetMacro(a.activeSchedule.MacroID)
			_ = a.store.AppendHistory(HistoryEntry{
				Time:       time.Now(),
				MacroID:    m.ID,
				MacroName:  m.Name,
				ScheduleID: a.activeSchedule.ID,
				SchedName:  a.activeSchedule.Name,
				Trigger:    "schedule",
				Result:     "aborted",
				Detail:     "Cancelled during pre-run countdown",
			})
			a.finalizeScheduleRun(a.activeSchedule.ID, "aborted")
			a.activeSchedule = nil
		}
		a.pushState()
	case statePlaying:
		a.requestAbort()
		a.note = "Aborting"
		a.pushState()
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

func (a *application) statusText() string {
	switch a.state {
	case stateRecording:
		n, _, elapsed, _ := a.rec.snapshot()
		return fmt.Sprintf("Recording · %d events · %s", n, formatLength(elapsed))
	case stateCountdown:
		remain := time.Until(a.countdownDeadline)
		if remain < 0 {
			remain = 0
		}
		if a.activeSchedule != nil {
			return fmt.Sprintf("Scheduled: %s in %s", a.activeSchedule.Name, formatRemain(remain))
		}
		return "Starting in " + formatRemain(remain)
	case statePlaying:
		if a.note != "" {
			return a.note
		}
		return "Playing macro…"
	default:
		macros, schedules, _, _ := a.store.Snapshot()
		activeSched := 0
		for _, sc := range schedules {
			if sc.Enabled {
				activeSched++
			}
		}
		base := fmt.Sprintf("Ready · %d macros · %d schedules active", len(macros), activeSched)
		if a.note != "" {
			return a.note + " · " + base
		}
		return base
	}
}

func (a *application) showMainWindow() {
	if a.hwnd == 0 {
		return
	}
	procShowWindow.Call(a.hwnd, swRestore)
	procShowWindow.Call(a.hwnd, swShow)
	procSetForegroundWindow.Call(a.hwnd)
}

// --- Scheduler Engine ---

func (a *application) wakeScheduler() {
	a.applyKeepAwake()
	select {
	case a.schedWake <- struct{}{}:
	default:
	}
}

func (a *application) schedulerLoop() {
	for {
		waitDur := a.nextScheduleSleepDuration()
		var timer *time.Timer
		var timerCh <-chan time.Time
		if waitDur >= 0 {
			timer = time.NewTimer(waitDur)
			timerCh = timer.C
		}
		select {
		case <-a.schedStop:
			if timer != nil {
				timer.Stop()
			}
			return
		case <-a.schedWake:
			if timer != nil {
				timer.Stop()
			}
		case <-timerCh:
			postMessage(a.msgHwnd, wmAppScheduleFire, 0, 0)
		}
	}
}

func (a *application) nextScheduleSleepDuration() time.Duration {
	now := time.Now()
	var earliest *time.Time
	_ = a.store.UpdateSchedules(func(schedules []Schedule) []Schedule {
		for i := range schedules {
			sc := &schedules[i]
			if !sc.Enabled {
				sc.NextRun = nil
				continue
			}
			if sc.NextRun == nil {
				sc.NextRun = ComputeNextRun(*sc, now)
			}
			if sc.NextRun != nil {
				if earliest == nil || sc.NextRun.Before(*earliest) {
					t := *sc.NextRun
					earliest = &t
				}
			}
		}
		return schedules
	})
	if earliest == nil {
		return -1
	}
	d := time.Until(*earliest)
	if d < 0 {
		return 50 * time.Millisecond
	}
	return d
}

func (a *application) onSchedulerCheckDue() {
	now := time.Now()
	var due []Schedule
	_ = a.store.UpdateSchedules(func(schedules []Schedule) []Schedule {
		for i := range schedules {
			sc := &schedules[i]
			if !sc.Enabled || sc.NextRun == nil {
				continue
			}
			if !sc.NextRun.After(now) {
				// If it's more than 45 seconds late (e.g. after PC sleep) and policy is Skip, log missed and advance.
				if now.Sub(*sc.NextRun) > 45*time.Second && sc.Missed == MissedSkip {
					sc.LastResult = "missed"
					sc.NextRun = ComputeNextRun(*sc, now)
					if sc.NextRun == nil {
						sc.Enabled = false
					}
					m, _ := a.store.GetMacro(sc.MacroID)
					_ = a.store.AppendHistory(HistoryEntry{
						Time:       now,
						MacroID:    sc.MacroID,
						MacroName:  m.Name,
						ScheduleID: sc.ID,
						SchedName:  sc.Name,
						Trigger:    "schedule",
						Result:     "missed",
						Detail:     "Missed while system was asleep or busy",
					})
					continue
				}
				due = append(due, *sc)
				// Advance NextRun tentatively so we don't double-fire while the run is in progress
				sc.NextRun = ComputeNextRun(*sc, now.Add(time.Second))
			}
		}
		return schedules
	})

	for _, sc := range due {
		a.triggerScheduledMacro(sc)
	}
	a.wakeScheduler()
	a.pushState()
}

func (a *application) triggerScheduledMacro(sc Schedule) {
	m, ok := a.store.GetMacro(sc.MacroID)
	if !ok || len(m.Steps) == 0 {
		a.finalizeScheduleRun(sc.ID, "skipped")
		return
	}
	// Never interrupt an active user recording
	if a.state == stateRecording {
		_ = a.store.AppendHistory(HistoryEntry{
			Time:       time.Now(),
			MacroID:    m.ID,
			MacroName:  m.Name,
			ScheduleID: sc.ID,
			SchedName:  sc.Name,
			Trigger:    "schedule",
			Result:     "skipped",
			Detail:     "Skipped because recording was active",
		})
		a.finalizeScheduleRun(sc.ID, "skipped")
		return
	}
	// Check locked desktop
	if !isDesktopUnlocked() {
		_ = a.store.AppendHistory(HistoryEntry{
			Time:       time.Now(),
			MacroID:    m.ID,
			MacroName:  m.Name,
			ScheduleID: sc.ID,
			SchedName:  sc.Name,
			Trigger:    "schedule",
			Result:     "skipped",
			Detail:     "Skipped because workstation was locked",
		})
		a.finalizeScheduleRun(sc.ID, "skipped")
		return
	}
	// Check overlap with another playing/countdown macro
	if a.state != stateIdle {
		if sc.Overlap == OverlapQueue {
			a.runQueue = append(a.runQueue, queuedRun{sched: sc})
			a.pushToast("info", fmt.Sprintf("Queued scheduled run: %s", sc.Name))
		} else {
			_ = a.store.AppendHistory(HistoryEntry{
				Time:       time.Now(),
				MacroID:    m.ID,
				MacroName:  m.Name,
				ScheduleID: sc.ID,
				SchedName:  sc.Name,
				Trigger:    "schedule",
				Result:     "skipped",
				Detail:     "Skipped due to overlap with active playback",
			})
			a.finalizeScheduleRun(sc.ID, "skipped")
		}
		return
	}

	scCopy := sc
	a.activeSchedule = &scCopy
	a.activeMacroID = m.ID
	a.playFromStep = 0

	warnSecs := a.store.GetSettings().ScheduleWarnSecs
	if warnSecs < 0 {
		warnSecs = 0
	}
	if warnSecs == 0 {
		a.beginPlayback()
		return
	}
	a.state = stateCountdown
	a.countdownDeadline = time.Now().Add(time.Duration(warnSecs) * time.Second)
	a.tray.showBalloon("Scheduled Macro Starting", fmt.Sprintf("%s will play in %ds (press Abort hotkey to cancel).", sc.Name, warnSecs))
	procSetTimer.Call(a.msgHwnd, timerCountdown, 100, 0)
	a.pushState()
}

func (a *application) finalizeScheduleRun(schedID string, result string) {
	now := time.Now()
	_ = a.store.UpdateSchedules(func(schedules []Schedule) []Schedule {
		for i := range schedules {
			sc := &schedules[i]
			if sc.ID != schedID {
				continue
			}
			sc.LastRun = &now
			sc.LastResult = result
			if result == "done" {
				sc.RunCount++
			}
			sc.NextRun = ComputeNextRun(*sc, now.Add(time.Second))
			if sc.NextRun == nil {
				sc.Enabled = false
			}
		}
		return schedules
	})
	a.wakeScheduler()
}

func (a *application) shutdown() {
	close(a.schedStop)
	setKeepAwake(false)
	a.tray.remove()
	if a.msgHwnd != 0 {
		procKillTimer.Call(a.msgHwnd, timerRecord)
		procKillTimer.Call(a.msgHwnd, timerCountdown)
		for _, id := range a.hotkeyIDs {
			procUnregisterHotKey.Call(a.msgHwnd, id)
		}
		procDestroyWindow.Call(a.msgHwnd)
		a.msgHwnd = 0
	}
	a.requestAbort()
	a.rec.stop()
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
