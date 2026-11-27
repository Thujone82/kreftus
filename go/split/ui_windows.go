//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

// UISnapshot is the complete state sent to the WebView2 frontend.
type UISnapshot struct {
	Theme           string             `json:"theme"`
	Mode            string             `json:"mode"`
	SkipDropped     bool               `json:"skipDropped"`
	ShowAudio       bool               `json:"showAudio"`
	ShowVisual      bool               `json:"showVisual"`
	TimelineScale   float64            `json:"timelineScale"`
	Tools           ToolStatus         `json:"tools"`
	Loading         bool               `json:"loading"`
	LoadingFile     string             `json:"loadingFile"`
	LoadingPct      float64            `json:"loadingPct"`
	LoadingPhase    string             `json:"loadingPhase"`
	LoadError       string             `json:"loadError"`
	LoadSeq         int64              `json:"loadSeq"`
	FilePath        string             `json:"filePath"`
	VideoURL        string             `json:"videoUrl"`
	RemuxURL        string             `json:"remuxUrl"`
	ScrubURL        string             `json:"scrubUrl"`
	Media           *MediaInfo         `json:"media"`
	Waveform         *AudioWaveform     `json:"waveform,omitempty"`
	WaveformLoading  bool               `json:"waveformLoading"`
	Visual           *VisualDelta       `json:"visual,omitempty"`
	VisualLoading    bool               `json:"visualLoading"`
	KeyframesLoading bool               `json:"keyframesLoading"`
	Markers          []float64          `json:"markers"`
	Segments        []Segment          `json:"segments"`
	OutputPreview   []string           `json:"outputPreview"`
	Exporting       bool               `json:"exporting"`
	ExportProgress  *ExportProgress    `json:"exportProgress,omitempty"`
	LastExport      *ExportResult      `json:"lastExport,omitempty"`
	LastExportSec   float64            `json:"lastExportSec,omitempty"`
}

type MarkerActionResult struct {
	SnappedSec float64    `json:"snappedSec"`
	Changed    bool       `json:"changed"`
	Snapshot   UISnapshot `json:"snapshot"`
}

type splitApp struct {
	hwnd       uintptr
	mainThread uint32
	chromium   *edge.Chromium
	srv        *mediaServer
	settings   *SettingsStore

	dispMu    sync.Mutex
	dispatchQ []func()

	mu               sync.Mutex
	tools            ToolStatus
	proj             *ProjectState
	waveform         *AudioWaveform
	waveformLoading  bool
	visual           *VisualDelta
	visualLoading    bool
	keyframesLoading bool
	loading          bool
	loadingFile      string
	loadingPct       float64
	loadingPhase     string
	loadError        string
	loadSeq          int64
	probeCancel      context.CancelFunc
	loadCtx          context.Context
	loadCancel       context.CancelFunc
	audioCancel      context.CancelFunc
	audioRunID       int64
	visualCancel     context.CancelFunc
	visualRunID      int64
	keyframesCancel  context.CancelFunc
	keyframesRunID   int64
	exporting        bool
	exportProg      *ExportProgress
	exportCancel    context.CancelFunc
	lastExport      *ExportResult
	lastExportSec   float64
	bindings        map[string]any
	dropMsgHandler  *fileDropWebMessageHandler
}

var (
	activeSplitApp *splitApp
	splitWndMap    = map[uintptr]*splitApp{}
	splitWndMu     sync.RWMutex
	splitWndProcCb uintptr

	iidUnknown                    = edge.NewGUID("{00000000-0000-0000-C000-000000000046}")
	iidWebMsgReceivedHandler      = edge.NewGUID("{57213f19-00e6-49fa-8e07-898ea01ecbd2}")
	iidWebMsgReceivedEventArgs2   = edge.NewGUID("{06fc7ab7-c90c-4297-9389-33ca01cf6d5e}")
	iidCoreWebView2File           = edge.NewGUID("{f2c19559-6bc1-4583-a757-90021be9afec}")
)

type fileDropWebMessageHandlerVtbl struct {
	QueryInterface edge.ComProc
	AddRef         edge.ComProc
	Release        edge.ComProc
	Invoke         edge.ComProc
}

type fileDropWebMessageHandler struct {
	vtbl *fileDropWebMessageHandlerVtbl
}

var fileDropHandlerFn = fileDropWebMessageHandlerVtbl{
	QueryInterface: edge.NewComProc(fileDropQueryInterface),
	AddRef:         edge.NewComProc(fileDropAddRef),
	Release:        edge.NewComProc(fileDropRelease),
	Invoke:         edge.NewComProc(fileDropInvoke),
}

func newFileDropWebMessageHandler() *fileDropWebMessageHandler {
	return &fileDropWebMessageHandler{vtbl: &fileDropHandlerFn}
}

func fileDropQueryInterface(this, refiid, ppvObject uintptr) uintptr {
	const eNoInterface = 0x80004002
	if ppvObject == 0 {
		return eNoInterface
	}
	if refiid != 0 {
		g := (*edge.GUID)(unsafe.Pointer(refiid))
		if edge.IsEqualGUID(g, iidUnknown) || edge.IsEqualGUID(g, iidWebMsgReceivedHandler) {
			*(*uintptr)(unsafe.Pointer(ppvObject)) = this
			return 0
		}
	}
	*(*uintptr)(unsafe.Pointer(ppvObject)) = 0
	return eNoInterface
}

func fileDropAddRef(_ uintptr) uintptr { return 1 }

func fileDropRelease(_ uintptr) uintptr { return 1 }

func comRelease(obj uintptr) {
	if obj == 0 {
		return
	}
	vtbl := (*[3]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
	_, _, _ = vtbl[2].Call(obj)
}

func comQueryInterface(obj uintptr, iid *edge.GUID) uintptr {
	if obj == 0 || iid == nil {
		return 0
	}
	var out uintptr
	vtbl := (*[3]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
	_, _, _ = vtbl[0].Call(obj, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	return out
}

func fileDropInvoke(_, _, args uintptr) uintptr {
	if args == 0 {
		return 0
	}
	argsVtbl := (*[6]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(args))))
	var msgPtr *uint16
	_, _, _ = argsVtbl[5].Call(args, uintptr(unsafe.Pointer(&msgPtr)))
	if msgPtr == nil {
		return 0
	}
	msg := windows.UTF16PtrToString(msgPtr)
	windows.CoTaskMemFree(unsafe.Pointer(msgPtr))
	if msg != "__file_drop__" {
		return 0
	}

	args2 := comQueryInterface(args, iidWebMsgReceivedEventArgs2)
	if args2 == 0 {
		return 0
	}
	defer comRelease(args2)

	args2Vtbl := (*[7]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(args2))))
	var coll uintptr
	_, _, _ = args2Vtbl[6].Call(args2, uintptr(unsafe.Pointer(&coll)))
	if coll == 0 {
		return 0
	}
	defer comRelease(coll)

	collVtbl := (*[5]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(coll))))
	var count uint32
	_, _, _ = collVtbl[3].Call(coll, uintptr(unsafe.Pointer(&count)))

	for i := uint32(0); i < count; i++ {
		var item uintptr
		_, _, _ = collVtbl[4].Call(coll, uintptr(i), uintptr(unsafe.Pointer(&item)))
		if item == 0 {
			continue
		}
		fileObj := comQueryInterface(item, iidCoreWebView2File)
		comRelease(item)
		if fileObj == 0 {
			continue
		}
		fileVtbl := (*[4]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(fileObj))))
		var pathPtr *uint16
		_, _, _ = fileVtbl[3].Call(fileObj, uintptr(unsafe.Pointer(&pathPtr)))
		comRelease(fileObj)
		if pathPtr != nil {
			droppedPath := windows.UTF16PtrToString(pathPtr)
			windows.CoTaskMemFree(unsafe.Pointer(pathPtr))
			if droppedPath != "" && activeSplitApp != nil {
				activeSplitApp.loadFileAsync(droppedPath)
				break
			}
		}
	}
	return 0
}

func (a *splitApp) registerFileDropHandler(ctrl *edge.ICoreWebView2Controller) {
	if ctrl == nil {
		return
	}
	a.dropMsgHandler = newFileDropWebMessageHandler()
	ctrlVtbl := (*[26]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(ctrl))))
	var wv2 uintptr
	_, _, _ = ctrlVtbl[25].Call(uintptr(unsafe.Pointer(ctrl)), uintptr(unsafe.Pointer(&wv2)))
	if wv2 == 0 {
		return
	}
	defer comRelease(wv2)

	wv2Vtbl := (*[35]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(wv2))))
	var tok int64
	_, _, _ = wv2Vtbl[34].Call(wv2, uintptr(unsafe.Pointer(a.dropMsgHandler)), uintptr(unsafe.Pointer(&tok)))
}

func runGUI(preloadPath string) error {
	runtime.LockOSThread()
	enablePerMonitorDPI()

	settings := NewSettingsStore()
	cfg := settings.Get()
	tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)

	srv, err := startMediaServer(tools.FFmpegPath)
	if err != nil {
		return fmt.Errorf("failed to start local media server: %w", err)
	}
	defer srv.Close()

	tid, _, _ := procGetCurrentThreadID.Call()
	app := &splitApp{
		mainThread: uint32(tid),
		srv:        srv,
		settings:   settings,
		tools:      tools,
		bindings:   make(map[string]any),
	}
	activeSplitApp = app

	wvDataDir := filepath.Join(os.Getenv("APPDATA"), "Split", "webview2")
	_ = os.MkdirAll(wvDataDir, 0o755)

	chromium := edge.NewChromium()
	chromium.DataPath = wvDataDir
	chromium.MessageCallback = app.onWebMessage
	chromium.SetPermission(edge.CoreWebView2PermissionKindClipboardRead, edge.CoreWebView2PermissionStateAllow)
	app.chromium = chromium

	if err := app.createHostWindow("Split", 1240, 800); err != nil {
		return err
	}
	if !chromium.Embed(app.hwnd) {
		return errors.New("failed to initialize Microsoft Edge WebView2 Runtime.\n\nInstall it via PowerShell:\nwinget install Microsoft.EdgeWebView2Runtime")
	}
	chromium.Resize()

	if wvSettings, err := chromium.GetSettings(); err == nil && wvSettings != nil {
		_ = wvSettings.PutAreDefaultContextMenusEnabled(false)
		_ = wvSettings.PutAreDevToolsEnabled(false)
	}

	app.applyThemeNative(cfg.Theme)
	app.registerFileDropHandler(chromium.GetController())

	// Allow WM_DROPFILES onto our outer frame/titlebar even if elevation differs
	if procChangeWindowMessageFilterEx.Find() == nil {
		_, _, _ = procChangeWindowMessageFilterEx.Call(app.hwnd, wmDropFiles, msgfltAllow, 0)
		_, _, _ = procChangeWindowMessageFilterEx.Call(app.hwnd, wmCopyGlobalData, msgfltAllow, 0)
	}
	_, _, _ = procDragAcceptFiles.Call(app.hwnd, 1)

	app.registerBindings()
	chromium.Navigate(srv.IndexURL())

	if preloadPath != "" {
		app.loadFileAsync(preloadPath)
	}

	app.messageLoop()
	return nil
}

func (a *splitApp) createHostWindow(title string, width, height uint) error {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	iconBig := loadSplitIcon(hInst, iconBig)
	iconSm := loadSplitIcon(hInst, iconSmall)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)

	if splitWndProcCb == 0 {
		splitWndProcCb = syscall.NewCallback(splitWndProc)
	}

	className := utf16Ptr("SplitMainWindow")
	wc := wndClassExW{
		CbSize:      uint32(unsafe.Sizeof(wndClassExW{})),
		LpfnWndProc: splitWndProcCb,
		HInstance:   hInst,
		HIcon:       iconBig,
		HIconSm:     iconSm,
		HCursor:     cursor,
		LpszClassName: className,
	}
	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)

	screenW, _, _ := procGetSystemMetrics.Call(smCxScreen)
	screenH, _, _ := procGetSystemMetrics.Call(smCyScreen)
	posX := (uint(screenW) - width) / 2
	posY := (uint(screenH) - height) / 2
	if int(screenW) <= int(width) {
		posX = 40
	}
	if int(screenH) <= int(height) {
		posY = 40
	}

	titlePtr := utf16Ptr(title)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsOverlappedWindow,
		uintptr(posX),
		uintptr(posY),
		uintptr(width),
		uintptr(height),
		0, 0, hInst, 0,
	)
	runtime.KeepAlive(className)
	runtime.KeepAlive(titlePtr)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}

	a.hwnd = hwnd
	splitWndMu.Lock()
	splitWndMap[hwnd] = a
	splitWndMu.Unlock()

	_, _, _ = procShowWindow.Call(hwnd, swShow)
	_, _, _ = procUpdateWindow.Call(hwnd)
	_, _, _ = procSetFocus.Call(hwnd)
	return nil
}

func loadSplitIcon(hInst uintptr, kind uintptr) uintptr {
	cxMetric := uintptr(smCxIcon)
	cyMetric := uintptr(smCyIcon)
	if kind == iconSmall {
		cxMetric = smCxSmIcon
		cyMetric = smCySmIcon
	}
	cx, _, _ := procGetSystemMetrics.Call(cxMetric)
	cy, _, _ := procGetSystemMetrics.Call(cyMetric)
	name := utf16Ptr("MAINICON")
	h, _, _ := procLoadImageW.Call(hInst, uintptr(unsafe.Pointer(name)), imageIcon, cx, cy, lrShared)
	runtime.KeepAlive(name)
	if h != 0 {
		return h
	}
	h, _, _ = procLoadIconW.Call(0, idiApplication)
	return h
}

func splitWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	splitWndMu.RLock()
	a := splitWndMap[hwnd]
	splitWndMu.RUnlock()

	if a != nil {
		switch msg {
		case wmMove, wmMoving:
			if a.chromium != nil {
				_ = a.chromium.NotifyParentWindowPositionChanged()
			}
			return 0
		case wmNCLButtonDown:
			_, _, _ = procSetFocus.Call(a.hwnd)
			r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
			return r
		case wmSize:
			if a.chromium != nil {
				a.chromium.Resize()
			}
			return 0
		case wmActivate:
			if wp != waInactive && a.chromium != nil {
				a.chromium.Focus()
			}
			return 0
		case wmDropFiles:
			a.handleDropFiles(wp)
			return 0
		case wmGetMinMaxInfo:
			mmi := (*minMaxInfo)(unsafe.Pointer(lp))
			mmi.PtMinTrackSize = winPoint{X: 940, Y: 620}
			return 0
		case wmClose:
			_, _, _ = procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			a.cancelLoadAndExport()
			_, _, _ = procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func (a *splitApp) handleDropFiles(hDrop uintptr) {
	defer procDragFinish.Call(hDrop)
	count, _, _ := procDragQueryFileW.Call(hDrop, 0xFFFFFFFF, 0, 0)
	if count == 0 {
		return
	}
	nChars, _, _ := procDragQueryFileW.Call(hDrop, 0, 0, 0)
	if nChars == 0 {
		return
	}
	buf := make([]uint16, nChars+1)
	_, _, _ = procDragQueryFileW.Call(hDrop, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	path := syscall.UTF16ToString(buf)
	if path != "" {
		a.loadFileAsync(path)
	}
}

func (a *splitApp) messageLoop() {
	var msg winMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		if msg.Message == wmApp {
			a.dispMu.Lock()
			q := append([]func(){}, a.dispatchQ...)
			a.dispatchQ = nil
			a.dispMu.Unlock()
			for _, fn := range q {
				fn()
			}
			continue
		}
		if msg.Message == wmQuit {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (a *splitApp) dispatchUI(fn func()) {
	a.dispMu.Lock()
	a.dispatchQ = append(a.dispatchQ, fn)
	a.dispMu.Unlock()
	_, _, _ = procPostThreadMessageW.Call(uintptr(a.mainThread), wmApp, 0, 0)
}

func (a *splitApp) runOnUIThreadSync(fn func()) {
	done := make(chan struct{})
	a.dispatchUI(func() {
		defer close(done)
		fn()
	})
	<-done
}

func (a *splitApp) evalJS(script string) {
	if a.chromium == nil {
		return
	}
	a.dispatchUI(func() {
		a.chromium.Eval(script)
	})
}

func (a *splitApp) pushState() {
	snap := a.snapshot()
	raw, err := json.Marshal(snap)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitState && window.__onSplitState(" + string(raw) + ");")
}

func (a *splitApp) pushProgress(p ExportProgress) {
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	a.evalJS("window.__onExportProgress && window.__onExportProgress(" + string(raw) + ");")
}

func (a *splitApp) pushWaveform(seq int64, wf *AudioWaveform) {
	payload := struct {
		Seq      int64          `json:"seq"`
		Waveform *AudioWaveform `json:"waveform"`
	}{Seq: seq, Waveform: wf}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitWaveform && window.__onSplitWaveform(" + string(raw) + ");")
}

func (a *splitApp) pushVisual(seq int64, vd *VisualDelta) {
	payload := struct {
		Seq    int64        `json:"seq"`
		Visual *VisualDelta `json:"visual"`
	}{Seq: seq, Visual: vd}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitVisual && window.__onSplitVisual(" + string(raw) + ");")
}

func (a *splitApp) pushLoadProgress(pct float64, kfs int, phase string) {
	payload := struct {
		Pct   float64 `json:"pct"`
		KFs   int     `json:"kfs"`
		Phase string  `json:"phase"`
	}{Pct: pct, KFs: kfs, Phase: phase}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onLoadProgress && window.__onLoadProgress(" + string(raw) + ");")
}

func (a *splitApp) pushLoadBackdrop(seq int64, dataURI string, kfSec float64) {
	payload := struct {
		Seq     int64   `json:"seq"`
		DataURI string  `json:"dataUri"`
		KFSec   float64 `json:"kfSec"`
	}{Seq: seq, DataURI: dataURI, KFSec: kfSec}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onLoadBackdrop && window.__onLoadBackdrop(" + string(raw) + ");")
}

func (a *splitApp) pushAnalysisProgress(seq int64, kind string, pct float64) {
	payload := struct {
		Seq  int64   `json:"seq"`
		Kind string  `json:"kind"`
		Pct  float64 `json:"pct"`
	}{Seq: seq, Kind: kind, Pct: pct}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onAnalysisProgress && window.__onAnalysisProgress(" + string(raw) + ");")
}

func (a *splitApp) applyThemeNative(theme string) {
	dark := theme != "light"
	setWindowDarkMode(a.hwnd, dark)
	if ctrl := a.chromium.GetController(); ctrl != nil {
		if ctrl2 := ctrl.GetICoreWebView2Controller2(); ctrl2 != nil {
			col := edge.COREWEBVIEW2_COLOR{A: 255, R: 15, G: 20, B: 29}
			if !dark {
				col = edge.COREWEBVIEW2_COLOR{A: 255, R: 241, G: 245, B: 249}
			}
			_ = ctrl2.PutDefaultBackgroundColor(col)
		}
	}
}

func (a *splitApp) cancelAnalysisWorkersLocked() {
	if a.keyframesCancel != nil {
		a.keyframesCancel()
		a.keyframesCancel = nil
	}
	if a.audioCancel != nil {
		a.audioCancel()
		a.audioCancel = nil
	}
	if a.visualCancel != nil {
		a.visualCancel()
		a.visualCancel = nil
	}
	a.keyframesLoading = false
	a.waveformLoading = false
	a.visualLoading = false
}

func (a *splitApp) cancelLoadAndExport() {
	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
		a.probeCancel = nil
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	if a.exportCancel != nil {
		a.exportCancel()
		a.exportCancel = nil
	}
	a.mu.Unlock()
}

func (a *splitApp) snapshot() UISnapshot {
	cfg := a.settings.Get()
	a.mu.Lock()
	defer a.mu.Unlock()

	snap := UISnapshot{
		Theme:            cfg.Theme,
		Mode:             cfg.ExportMode,
		SkipDropped:      cfg.SkipDropped,
		ShowAudio:        cfg.ShowAudio,
		ShowVisual:       cfg.ShowVisual,
		TimelineScale:    cfg.TimelineScale,
		Tools:            a.tools,
		Loading:          a.loading,
		LoadingFile:      a.loadingFile,
		LoadingPct:       a.loadingPct,
		LoadingPhase:     a.loadingPhase,
		LoadError:        a.loadError,
		LoadSeq:          a.loadSeq,
		Waveform:         a.waveform,
		WaveformLoading:  a.waveformLoading,
		Visual:           a.visual,
		VisualLoading:    a.visualLoading,
		KeyframesLoading: a.keyframesLoading,
		Exporting:        a.exporting,
		ExportProgress:   a.exportProg,
		LastExport:       a.lastExport,
		LastExportSec:    a.lastExportSec,
	}

	if a.proj != nil && a.proj.Media != nil {
		snap.Mode = a.proj.Mode
		snap.FilePath = a.proj.FilePath
		snap.Media = a.proj.Media
		snap.Markers = append([]float64(nil), a.proj.Markers...)
		snap.Segments = a.proj.Segments()
		seqStr := strconv.FormatInt(a.loadSeq, 10)
		snap.VideoURL = a.srv.baseURL + "/video?v=" + seqStr
		snap.RemuxURL = a.srv.baseURL + "/remux?v=" + seqStr
		snap.ScrubURL = a.srv.baseURL + "/scrub?v=" + seqStr

		kept := a.proj.KeptSegments()
		if len(kept) > 0 {
			if a.proj.Mode == "join" {
				snap.OutputPreview = []string{filepath.Base(SuggestedJoinPath(a.proj.FilePath))}
			} else {
				paths := DeriveSplitPaths(SuggestedSplitDialogPath(a.proj.FilePath), len(kept))
				for _, p := range paths {
					snap.OutputPreview = append(snap.OutputPreview, filepath.Base(p))
				}
			}
		}
	}
	return snap
}

func (a *splitApp) unloadMediaAndUnlock() {
	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
		a.probeCancel = nil
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	a.proj = nil
	a.waveform = nil
	a.visual = nil
	a.lastExport = nil
	a.lastExportSec = 0
	a.loadSeq++
	a.mu.Unlock()

	a.srv.ClearActiveVideo()
	a.dispatchUI(func() {
		t := utf16Ptr("Split")
		_, _, _ = procSetWindowTextW.Call(a.hwnd, uintptr(unsafe.Pointer(t)))
		runtime.KeepAlive(t)
	})
}

func (a *splitApp) loadFileAsync(rawPath string) {
	absPath, err := filepath.Abs(strings.TrimSpace(rawPath))
	if err != nil {
		absPath = rawPath
	}

	// Release any previously open file handles, background analysis, and active HTTP streams immediately.
	a.srv.ClearActiveVideo()

	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	probeCtx, probeCancel := context.WithCancel(context.Background())
	a.probeCancel = probeCancel
	a.proj = nil
	a.waveform = nil
	a.visual = nil
	a.loadSeq++
	probeSeq := a.loadSeq
	a.loading = true
	a.loadingFile = filepath.Base(absPath)
	a.loadingPct = 2
	a.loadingPhase = "Opening video…"
	a.loadError = ""
	a.lastExport = nil
	tools := FindMediaTools(a.settings.Get().FFmpegPath, a.settings.Get().FFprobePath)
	a.tools = tools
	a.mu.Unlock()

	a.dispatchUI(func() {
		t := utf16Ptr("Split")
		_, _, _ = procSetWindowTextW.Call(a.hwnd, uintptr(unsafe.Pointer(t)))
		runtime.KeepAlive(t)
	})
	a.srv.SetFFmpegPath(tools.FFmpegPath)
	a.pushState()

	go func() {
		defer probeCancel()
		if !tools.Ready {
			a.mu.Lock()
			a.loading = false
			a.loadingPct = 0
			a.loadingPhase = ""
			a.loadError = tools.Error
			a.mu.Unlock()
			a.pushState()
			return
		}

		var backdropBusy atomic.Bool
		var backdropMu sync.Mutex
		var lastBackdropAt time.Time
		lastBackdropKF := -1.0

		maybeEmitBackdrop := func(kfSec float64) {
			if probeCtx.Err() != nil {
				return
			}
			backdropMu.Lock()
			if !lastBackdropAt.IsZero() && time.Since(lastBackdropAt) < 1800*time.Millisecond {
				backdropMu.Unlock()
				return
			}
			if lastBackdropKF >= 0 && math.Abs(kfSec-lastBackdropKF) < 0.05 {
				backdropMu.Unlock()
				return
			}
			if !backdropBusy.CompareAndSwap(false, true) {
				backdropMu.Unlock()
				return
			}
			lastBackdropAt = time.Now()
			lastBackdropKF = kfSec
			backdropMu.Unlock()

			go func(targetKF float64) {
				defer backdropBusy.Store(false)
				uri, bErr := ExtractProbeBackdropDataURI(probeCtx, tools.FFmpegPath, absPath, targetKF)
				if bErr != nil || uri == "" || probeCtx.Err() != nil {
					return
				}
				a.mu.Lock()
				active := a.loading && a.loadSeq == probeSeq
				a.mu.Unlock()
				if active {
					a.pushLoadBackdrop(probeSeq, uri, targetKF)
				}
			}(kfSec)
		}

		maybeEmitBackdrop(0)

		media, err := ProbeMediaWithProgress(probeCtx, tools.FFprobePath, absPath, func(pct float64, kfs int, latestKFSec float64, phase string) {
			if probeCtx.Err() != nil {
				return
			}
			a.mu.Lock()
			stillLoading := a.loading
			if stillLoading {
				a.loadingPct = pct
				a.loadingPhase = phase
			}
			a.mu.Unlock()
			if stillLoading {
				a.pushLoadProgress(pct, kfs, phase)
				if kfs > 0 {
					maybeEmitBackdrop(latestKFSec)
				}
			}
		})
		if probeCtx.Err() != nil {
			return
		}
		if err != nil {
			a.mu.Lock()
			a.loading = false
			a.loadingPct = 0
			a.loadingPhase = ""
			a.loadError = err.Error()
			a.mu.Unlock()
			a.pushState()
			return
		}

		cfg := a.settings.Get()
		a.settings.Update(func(s *Settings) {
			s.LastDir = filepath.Dir(absPath)
		})
		a.srv.SetActiveVideo(absPath, media.VideoCodec)

		a.mu.Lock()
		a.cancelAnalysisWorkersLocked()
		if a.loadCancel != nil {
			a.loadCancel()
		}
		loadCtx, loadCancel := context.WithCancel(context.Background())
		a.loadCtx = loadCtx
		a.loadCancel = loadCancel
		a.probeCancel = nil
		a.loadSeq++
		a.loading = false
		a.loadingPct = 100
		a.loadingPhase = ""
		a.loadError = ""
		a.proj = NewProject(absPath, media, cfg.ExportMode)
		a.waveform = nil
		a.visual = nil

		var startAudio func()
		if !media.HasAudio {
			a.waveform = &AudioWaveform{Ready: true, HasAudio: false}
		} else if cfg.ShowAudio {
			startAudio = a.prepareAudioJobLocked()
		}
		var startVisual func()
		if cfg.ShowVisual {
			startVisual = a.prepareVisualJobLocked()
		}
		a.mu.Unlock()

		a.dispatchUI(func() {
			t := utf16Ptr("Split — " + media.FileName)
			_, _, _ = procSetWindowTextW.Call(a.hwnd, uintptr(unsafe.Pointer(t)))
			runtime.KeepAlive(t)
		})
		a.pushState()

		if startAudio != nil {
			go startAudio()
		}
		if startVisual != nil {
			go startVisual()
		}
	}()
}

// prepareAudioJobLocked cancels any prior audio analysis worker and prepares a fresh job. Caller holds a.mu.
func (a *splitApp) prepareAudioJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.proj.Media.HasAudio || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.audioCancel != nil {
		a.audioCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.audioCancel = cancel
	a.audioRunID++
	runID := a.audioRunID
	seq := a.loadSeq
	tools := a.tools
	absPath := a.proj.FilePath
	media := a.proj.Media
	a.waveform = nil
	a.waveformLoading = true
	return func() {
		defer cancel()
		a.startAudioAnalysis(ctx, seq, runID, tools, absPath, media)
	}
}

// prepareVisualJobLocked cancels any prior visual delta worker and prepares a fresh job. Caller holds a.mu.
func (a *splitApp) prepareVisualJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.visualCancel != nil {
		a.visualCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.visualCancel = cancel
	a.visualRunID++
	runID := a.visualRunID
	seq := a.loadSeq
	tools := a.tools
	media := a.proj.Media
	a.visual = nil
	a.visualLoading = true
	return func() {
		defer cancel()
		a.startVisualAnalysis(ctx, seq, runID, tools, media)
	}
}

// prepareKeyframesJobLocked cancels any prior keyframe refresh worker and prepares a non-destructive re-probe. Caller holds a.mu.
func (a *splitApp) prepareKeyframesJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.keyframesCancel != nil {
		a.keyframesCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.keyframesCancel = cancel
	a.keyframesRunID++
	runID := a.keyframesRunID
	seq := a.loadSeq
	tools := a.tools
	absPath := a.proj.FilePath
	a.keyframesLoading = true
	return func() {
		defer cancel()
		a.startKeyframesRefresh(ctx, seq, runID, tools, absPath)
	}
}

func (a *splitApp) startAudioAnalysis(ctx context.Context, seq, runID int64, tools ToolStatus, absPath string, media *MediaInfo) {
	wf, wfErr := ExtractAudioWaveformWithProgress(ctx, tools.FFmpegPath, absPath, media.DurationSec, media.HasAudio, func(pct float64) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.audioRunID == runID && a.waveformLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "audio", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	if wfErr != nil {
		wf = &AudioWaveform{Ready: true, HasAudio: false}
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.audioRunID == runID
	if stillCurrent {
		a.waveform = wf
		a.waveformLoading = false
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushWaveform(seq, wf)
	}
}

func (a *splitApp) startVisualAnalysis(ctx context.Context, seq, runID int64, tools ToolStatus, media *MediaInfo) {
	vd, vdErr := ExtractVisualDeltaWithProgress(ctx, tools.FFmpegPath, media, func(pct float64) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.visualRunID == runID && a.visualLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "visual", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	if vdErr != nil {
		vd = &VisualDelta{Ready: true}
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.visualRunID == runID
	if stillCurrent {
		a.visual = vd
		a.visualLoading = false
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushVisual(seq, vd)
	}
}

func (a *splitApp) startKeyframesRefresh(ctx context.Context, seq, runID int64, tools ToolStatus, absPath string) {
	media, err := ProbeMediaWithProgress(ctx, tools.FFprobePath, absPath, func(pct float64, kfs int, latestKFSec float64, phase string) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.keyframesRunID == runID && a.keyframesLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "keyframes", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.keyframesRunID == runID
	if stillCurrent {
		a.keyframesLoading = false
		if err == nil && media != nil && a.proj != nil {
			a.proj.UpdateMediaPreservingCuts(media)
		} else if err != nil {
			a.loadError = err.Error()
		}
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushState()
		if err != nil {
			a.mu.Lock()
			a.loadError = ""
			a.mu.Unlock()
		}
	}
}

// --- RPC Bridge ---

type rpcRequest struct {
	ID     int               `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func (a *splitApp) registerBindings() {
	a.bind("getState", func() (UISnapshot, error) {
		return a.snapshot(), nil
	})
	a.bind("openFileDialog", func() (bool, error) {
		cfg := a.settings.Get()
		var chosen string
		var ok bool
		a.runOnUIThreadSync(func() {
			chosen, ok = OpenVideoDialog(a.hwnd, cfg.LastDir)
		})
		if !ok || chosen == "" {
			return false, nil
		}
		a.loadFileAsync(chosen)
		return true, nil
	})
	a.bind("loadFile", func(path string) (bool, error) {
		if strings.TrimSpace(path) == "" {
			return false, nil
		}
		a.loadFileAsync(path)
		return true, nil
	})
	a.bind("cancelLoad", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.probeCancel != nil {
			a.probeCancel()
			a.probeCancel = nil
		}
		a.loading = false
		a.loadingPct = 0
		a.loadingPhase = ""
		a.loadError = ""
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("closeFile", func() (UISnapshot, error) {
		a.unloadMediaAndUnlock()
		a.mu.Lock()
		a.loading = false
		a.loadingPct = 0
		a.loadingPhase = ""
		a.loadError = ""
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("recheckTools", func() (UISnapshot, error) {
		cfg := a.settings.Get()
		tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
		a.mu.Lock()
		a.tools = tools
		a.mu.Unlock()
		a.srv.SetFFmpegPath(tools.FFmpegPath)
		return a.snapshot(), nil
	})
	a.bind("locateFFmpeg", func() (UISnapshot, error) {
		var chosen string
		var ok bool
		a.runOnUIThreadSync(func() {
			chosen, ok = OpenExeDialog(a.hwnd, "Locate ffmpeg.exe (or ffprobe.exe)", `C:\`)
		})
		if !ok || chosen == "" {
			return a.snapshot(), nil
		}
		dir := filepath.Dir(chosen)
		base := strings.ToLower(filepath.Base(chosen))
		a.settings.Update(func(s *Settings) {
			if strings.Contains(base, "ffprobe") {
				s.FFprobePath = chosen
				sib := filepath.Join(dir, "ffmpeg.exe")
				if st, err := os.Stat(sib); err == nil && !st.IsDir() {
					s.FFmpegPath = sib
				}
			} else {
				s.FFmpegPath = chosen
				sib := filepath.Join(dir, "ffprobe.exe")
				if st, err := os.Stat(sib); err == nil && !st.IsDir() {
					s.FFprobePath = sib
				}
			}
		})
		cfg := a.settings.Get()
		tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
		a.mu.Lock()
		a.tools = tools
		a.mu.Unlock()
		a.srv.SetFFmpegPath(tools.FFmpegPath)
		return a.snapshot(), nil
	})
	a.bind("addMarker", func(sec float64) (MarkerActionResult, error) {
		a.mu.Lock()
		if a.proj == nil {
			a.mu.Unlock()
			return MarkerActionResult{}, errors.New("no video loaded")
		}
		snapped, added := a.proj.AddMarkerAt(sec)
		a.mu.Unlock()
		return MarkerActionResult{
			SnappedSec: snapped,
			Changed:    added,
			Snapshot:   a.snapshot(),
		}, nil
	})
	a.bind("moveMarker", func(idx int, sec float64) (MarkerActionResult, error) {
		a.mu.Lock()
		if a.proj == nil {
			a.mu.Unlock()
			return MarkerActionResult{}, errors.New("no video loaded")
		}
		snapped, moved := a.proj.MoveMarker(idx, sec)
		a.mu.Unlock()
		return MarkerActionResult{
			SnappedSec: snapped,
			Changed:    moved,
			Snapshot:   a.snapshot(),
		}, nil
	})
	a.bind("removeMarker", func(idx int) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.RemoveMarker(idx)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("clearMarkers", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.ClearMarkers()
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("toggleSegment", func(segIdx int) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.ToggleSegment(segIdx)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setAllSegments", func(keep bool) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.SetAllKeep(keep)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("invertSegments", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.InvertKeep()
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setMode", func(mode string) (UISnapshot, error) {
		if mode != "split" {
			mode = "join"
		}
		a.settings.Update(func(s *Settings) { s.ExportMode = mode })
		a.mu.Lock()
		if a.proj != nil {
			a.proj.SetMode(mode)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setTheme", func(theme string) (UISnapshot, error) {
		if theme != "light" {
			theme = "dark"
		}
		a.settings.Update(func(s *Settings) { s.Theme = theme })
		a.runOnUIThreadSync(func() {
			a.applyThemeNative(theme)
		})
		return a.snapshot(), nil
	})
	a.bind("setSkipDropped", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.SkipDropped = enabled })
		return a.snapshot(), nil
	})
	a.bind("setShowAudio", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.ShowAudio = enabled })
		if enabled {
			a.mu.Lock()
			var startAudio func()
			if a.proj != nil && a.proj.Media != nil && a.proj.Media.HasAudio && a.waveform == nil && !a.waveformLoading {
				startAudio = a.prepareAudioJobLocked()
			}
			a.mu.Unlock()
			if startAudio != nil {
				go startAudio()
			}
		}
		return a.snapshot(), nil
	})
	a.bind("setShowVisual", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.ShowVisual = enabled })
		if enabled {
			a.mu.Lock()
			var startVisual func()
			if a.proj != nil && a.proj.Media != nil && a.visual == nil && !a.visualLoading {
				startVisual = a.prepareVisualJobLocked()
			}
			a.mu.Unlock()
			if startVisual != nil {
				go startVisual()
			}
		}
		return a.snapshot(), nil
	})
	a.bind("refreshAnalysis", func(target string) (UISnapshot, error) {
		cfg := a.settings.Get()
		if !a.tools.Ready {
			tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
			a.mu.Lock()
			a.tools = tools
			a.mu.Unlock()
		}

		wantKF := target == "keyframes" || target == "all"
		wantAudio := target == "audio" || target == "all"
		wantVisual := target == "visual" || target == "all"

		if wantAudio && !cfg.ShowAudio {
			a.settings.Update(func(s *Settings) { s.ShowAudio = true })
		}
		if wantVisual && !cfg.ShowVisual {
			a.settings.Update(func(s *Settings) { s.ShowVisual = true })
		}

		a.mu.Lock()
		var jobs []func()
		if wantKF {
			if j := a.prepareKeyframesJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		if wantAudio {
			if j := a.prepareAudioJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		if wantVisual {
			if j := a.prepareVisualJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		a.mu.Unlock()

		for _, j := range jobs {
			go j()
		}
		return a.snapshot(), nil
	})
	a.bind("setTimelineScale", func(scale float64) (UISnapshot, error) {
		clamped := ClampTimelineScale(scale)
		a.settings.Update(func(s *Settings) { s.TimelineScale = clamped })
		return a.snapshot(), nil
	})
	a.bind("exportVideo", func() (bool, error) {
		return a.startExportFlow()
	})
	a.bind("cancelExport", func() (bool, error) {
		a.mu.Lock()
		cancel := a.exportCancel
		a.mu.Unlock()
		if cancel != nil {
			cancel()
			return true, nil
		}
		return false, nil
	})
	a.bind("revealInExplorer", func(path string) (bool, error) {
		if path == "" {
			return false, nil
		}
		cmd := exec.Command("explorer.exe", "/select,"+path)
		hideSubprocessWindow(cmd)
		_ = cmd.Start()
		return true, nil
	})

	a.chromium.Init(`
		window.external = { invoke: s => window.chrome.webview.postMessage(s) };
		(function() {
			const RPC = window._rpc = (window._rpc || { nextSeq: 1 });
			const methods = [
				"getState", "openFileDialog", "loadFile", "cancelLoad", "closeFile",
				"recheckTools", "locateFFmpeg",
				"addMarker", "moveMarker", "removeMarker", "clearMarkers",
				"toggleSegment", "setAllSegments", "invertSegments",
				"setMode", "setTheme", "setSkipDropped", "setShowAudio", "setShowVisual", "setTimelineScale",
				"refreshAnalysis", "exportVideo", "cancelExport", "revealInExplorer"
			];
			window.api = {};
			for (const name of methods) {
				window.api[name] = function() {
					const seq = RPC.nextSeq++;
					const promise = new Promise((resolve, reject) => {
						RPC[seq] = { resolve, reject };
					});
					window.external.invoke(JSON.stringify({
						id: seq,
						method: name,
						params: Array.prototype.slice.call(arguments)
					}));
					return promise;
				};
			}
		})();
	`)
}

func (a *splitApp) bind(name string, fn any) {
	a.bindings[name] = fn
}

func (a *splitApp) onWebMessage(raw string) {
	if raw == "__file_drop__" {
		return
	}
	var req rpcRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return
	}
	fn, ok := a.bindings[req.Method]
	if !ok {
		a.resolveRPC(req.ID, nil, fmt.Errorf("unknown method %q", req.Method))
		return
	}

	go func() {
		res, err := invokeBinding(fn, req.Params)
		a.resolveRPC(req.ID, res, err)
	}()
}

func (a *splitApp) resolveRPC(id int, res any, err error) {
	idStr := strconv.Itoa(id)
	if err != nil {
		msgJSON, _ := json.Marshal(err.Error())
		a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].reject(" + string(msgJSON) + "); delete window._rpc[" + idStr + "];")
		return
	}
	b, mErr := json.Marshal(res)
	if mErr != nil {
		msgJSON, _ := json.Marshal(mErr.Error())
		a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].reject(" + string(msgJSON) + "); delete window._rpc[" + idStr + "];")
		return
	}
	a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].resolve(" + string(b) + "); delete window._rpc[" + idStr + "];")
}

func invokeBinding(fn any, rawParams []json.RawMessage) (any, error) {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if len(rawParams) != t.NumIn() {
		return nil, fmt.Errorf("expected %d arguments, got %d", t.NumIn(), len(rawParams))
	}
	args := make([]reflect.Value, len(rawParams))
	for i, p := range rawParams {
		argPtr := reflect.New(t.In(i))
		if err := json.Unmarshal(p, argPtr.Interface()); err != nil {
			return nil, err
		}
		args[i] = argPtr.Elem()
	}
	out := v.Call(args)
	if len(out) == 2 {
		if !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		}
		return out[0].Interface(), nil
	}
	if len(out) == 1 {
		return out[0].Interface(), nil
	}
	return nil, nil
}

func (a *splitApp) startExportFlow() (bool, error) {
	a.mu.Lock()
	if a.exporting {
		a.mu.Unlock()
		return false, errors.New("an export is already in progress")
	}
	if a.proj == nil || a.proj.Media == nil {
		a.mu.Unlock()
		return false, errors.New("load a video file first")
	}
	if !a.tools.Ready {
		errMsg := a.tools.Error
		a.mu.Unlock()
		return false, errors.New(errMsg)
	}
	kept := a.proj.KeptSegments()
	if len(kept) == 0 {
		a.mu.Unlock()
		return false, errors.New("select at least one segment as Keep before exporting")
	}

	// Snapshot project copy so UI edits during export cannot corrupt in-flight export
	projCopy := &ProjectState{
		FilePath:  a.proj.FilePath,
		Media:     a.proj.Media,
		Markers:   append([]float64(nil), a.proj.Markers...),
		KeepFlags: append([]bool(nil), a.proj.KeepFlags...),
		Mode:      a.proj.Mode,
	}
	tools := a.tools
	a.mu.Unlock()

	srcDir, _, ext := SplitBaseExt(projCopy.FilePath)
	var chosen string
	var ok bool
	var splitTargets []string

	a.runOnUIThreadSync(func() {
		if projCopy.Mode == "join" {
			defName := filepath.Base(SuggestedJoinPath(projCopy.FilePath))
			chosen, ok = SaveVideoDialog(a.hwnd, "Export Joined Video", srcDir, defName, ext, true)
		} else {
			defName := filepath.Base(SuggestedSplitDialogPath(projCopy.FilePath))
			chosen, ok = SaveVideoDialog(a.hwnd, fmt.Sprintf("Export %d Split Segments (Base Name)", len(kept)), srcDir, defName, ext, false)
			if ok && chosen != "" {
				splitTargets = DeriveSplitPaths(chosen, len(kept))
				var existing []string
				for _, tp := range splitTargets {
					if _, err := os.Stat(tp); err == nil {
						existing = append(existing, filepath.Base(tp))
					}
				}
				if len(existing) > 0 {
					preview := strings.Join(existing, ", ")
					if len(existing) > 5 {
						preview = strings.Join(existing[:5], ", ") + fmt.Sprintf(" (+%d more)", len(existing)-5)
					}
					if !confirmOverwriteDialog(a.hwnd, "Overwrite Existing Files?", fmt.Sprintf("The following output file(s) already exist:\n\n%s\n\nOverwrite them?", preview)) {
						ok = false
					}
				}
			}
		}
	})

	if !ok || chosen == "" {
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.exporting = true
	a.exportCancel = cancel
	a.lastExport = nil
	a.exportProg = &ExportProgress{
		Current: 0,
		Total:   len(kept),
		Label:   filepath.Base(chosen),
		Phase:   "Starting lossless export...",
	}
	a.mu.Unlock()
	a.pushState()

	go func() {
		defer cancel()
		sw := time.Now()
		onProg := func(p ExportProgress) {
			a.mu.Lock()
			pc := p
			a.exportProg = &pc
			a.mu.Unlock()
			a.pushProgress(p)
		}

		var res *ExportResult
		var err error
		if projCopy.Mode == "join" {
			res, err = ExportJoin(ctx, tools, projCopy, chosen, onProg)
		} else {
			res, err = ExportSplit(ctx, tools, projCopy, splitTargets, onProg)
		}

		elapsed := time.Since(sw).Seconds()
		a.mu.Lock()
		a.exporting = false
		a.exportCancel = nil
		a.exportProg = nil
		if err == nil {
			a.lastExport = res
			a.lastExportSec = elapsed
		}
		a.mu.Unlock()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				a.evalJS("window.__onExportDone && window.__onExportDone({cancelled:true});")
			} else {
				errJSON, _ := json.Marshal(err.Error())
				a.evalJS("window.__onExportDone && window.__onExportDone({error:" + string(errJSON) + "});")
			}
			a.pushState()
			return
		}

		a.pushState()
		a.evalJS("window.__onExportDone && window.__onExportDone({ok:true});")
	}()

	return true, nil
}

// Keep unused windows import clean
var _ = windows.ERROR_SUCCESS
