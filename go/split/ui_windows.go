//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

type winHost struct {
	hwnd       uintptr
	mainThread uint32
	chromium   *edge.Chromium
	app        *splitApp

	dispMu         sync.Mutex
	dispatchQ      []func()
	dropMsgHandler *fileDropWebMessageHandler
}

var (
	activeWinHost *winHost
	splitWndMap   = map[uintptr]*winHost{}
	splitWndMu    sync.RWMutex
	splitWndProcCb uintptr

	iidUnknown                  = edge.NewGUID("{00000000-0000-0000-C000-000000000046}")
	iidWebMsgReceivedHandler    = edge.NewGUID("{57213f19-00e6-49fa-8e07-898ea01ecbd2}")
	iidWebMsgReceivedEventArgs2 = edge.NewGUID("{06fc7ab7-c90c-4297-9389-33ca01cf6d5e}")
	iidCoreWebView2File         = edge.NewGUID("{f2c19559-6bc1-4583-a757-90021be9afec}")
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
			if droppedPath != "" && activeWinHost != nil && activeWinHost.app != nil {
				activeWinHost.app.loadFileAsync(droppedPath)
				break
			}
		}
	}
	return 0
}

func (h *winHost) registerFileDropHandler(ctrl *edge.ICoreWebView2Controller) {
	if ctrl == nil {
		return
	}
	h.dropMsgHandler = newFileDropWebMessageHandler()
	ctrlVtbl := (*[26]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(ctrl))))
	var wv2 uintptr
	_, _, _ = ctrlVtbl[25].Call(uintptr(unsafe.Pointer(ctrl)), uintptr(unsafe.Pointer(&wv2)))
	if wv2 == 0 {
		return
	}
	defer comRelease(wv2)

	wv2Vtbl := (*[35]edge.ComProc)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(wv2))))
	var tok int64
	_, _, _ = wv2Vtbl[34].Call(wv2, uintptr(unsafe.Pointer(h.dropMsgHandler)), uintptr(unsafe.Pointer(&tok)))
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
	host := &winHost{
		mainThread: uint32(tid),
	}
	app := newSplitApp(srv, settings, tools, host)
	host.app = app
	activeWinHost = host

	wvDataDir := filepath.Join(os.Getenv("APPDATA"), "Split", "webview2")
	_ = os.MkdirAll(wvDataDir, 0o755)

	chromium := edge.NewChromium()
	chromium.DataPath = wvDataDir
	chromium.MessageCallback = app.onWebMessage
	chromium.SetPermission(edge.CoreWebView2PermissionKindClipboardRead, edge.CoreWebView2PermissionStateAllow)
	host.chromium = chromium

	if err := host.createHostWindow("Split", 1240, 800); err != nil {
		return err
	}
	if !chromium.Embed(host.hwnd) {
		return errors.New("failed to initialize Microsoft Edge WebView2 Runtime.\n\nInstall it via PowerShell:\nwinget install Microsoft.EdgeWebView2Runtime")
	}
	chromium.Resize()

	if wvSettings, err := chromium.GetSettings(); err == nil && wvSettings != nil {
		_ = wvSettings.PutAreDefaultContextMenusEnabled(false)
		_ = wvSettings.PutAreDevToolsEnabled(false)
	}

	host.ApplyTheme(cfg.Theme)
	host.registerFileDropHandler(chromium.GetController())

	// Allow WM_DROPFILES onto our outer frame/titlebar even if elevation differs
	if procChangeWindowMessageFilterEx.Find() == nil {
		_, _, _ = procChangeWindowMessageFilterEx.Call(host.hwnd, wmDropFiles, msgfltAllow, 0)
		_, _, _ = procChangeWindowMessageFilterEx.Call(host.hwnd, wmCopyGlobalData, msgfltAllow, 0)
	}
	_, _, _ = procDragAcceptFiles.Call(host.hwnd, 1)

	chromium.Init(`
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
	chromium.Navigate(srv.IndexURL())

	if preloadPath != "" {
		app.loadFileAsync(preloadPath)
	}

	host.messageLoop()
	return nil
}

func (h *winHost) createHostWindow(title string, width, height uint) error {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	iconBig := loadSplitIcon(hInst, iconBig)
	iconSm := loadSplitIcon(hInst, iconSmall)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)

	if splitWndProcCb == 0 {
		splitWndProcCb = syscall.NewCallback(splitWndProc)
	}

	className := utf16Ptr("SplitMainWindow")
	wc := wndClassExW{
		CbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		LpfnWndProc:   splitWndProcCb,
		HInstance:     hInst,
		HIcon:         iconBig,
		HIconSm:       iconSm,
		HCursor:       cursor,
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

	h.hwnd = hwnd
	splitWndMu.Lock()
	splitWndMap[hwnd] = h
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
	h := splitWndMap[hwnd]
	splitWndMu.RUnlock()

	if h != nil {
		switch msg {
		case wmMove, wmMoving:
			if h.chromium != nil {
				_ = h.chromium.NotifyParentWindowPositionChanged()
			}
			return 0
		case wmNCLButtonDown:
			_, _, _ = procSetFocus.Call(h.hwnd)
			r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
			return r
		case wmSize:
			if h.chromium != nil {
				h.chromium.Resize()
			}
			return 0
		case wmActivate:
			if wp != waInactive && h.chromium != nil {
				h.chromium.Focus()
			}
			return 0
		case wmDropFiles:
			h.handleDropFiles(wp)
			return 0
		case wmGetMinMaxInfo:
			mmi := (*minMaxInfo)(unsafe.Pointer(lp))
			mmi.PtMinTrackSize = winPoint{X: 940, Y: 620}
			return 0
		case wmClose:
			_, _, _ = procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			if h.app != nil {
				h.app.cancelLoadAndExport()
			}
			_, _, _ = procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func (h *winHost) handleDropFiles(hDrop uintptr) {
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
	if path != "" && h.app != nil {
		h.app.loadFileAsync(path)
	}
}

func (h *winHost) messageLoop() {
	var msg winMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		if msg.Message == wmApp {
			h.dispMu.Lock()
			q := append([]func(){}, h.dispatchQ...)
			h.dispatchQ = nil
			h.dispMu.Unlock()
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

func (h *winHost) DispatchUI(fn func()) {
	h.dispMu.Lock()
	h.dispatchQ = append(h.dispatchQ, fn)
	h.dispMu.Unlock()
	_, _, _ = procPostThreadMessageW.Call(uintptr(h.mainThread), wmApp, 0, 0)
}

func (h *winHost) EvalJS(script string) {
	if h.chromium == nil {
		return
	}
	h.DispatchUI(func() {
		h.chromium.Eval(script)
	})
}

func (h *winHost) SetWindowTitle(title string) {
	h.DispatchUI(func() {
		t := utf16Ptr(title)
		_, _, _ = procSetWindowTextW.Call(h.hwnd, uintptr(unsafe.Pointer(t)))
		runtime.KeepAlive(t)
	})
}

func (h *winHost) ApplyTheme(theme string) {
	dark := theme != "light"
	setWindowDarkMode(h.hwnd, dark)
	if h.chromium == nil {
		return
	}
	if ctrl := h.chromium.GetController(); ctrl != nil {
		if ctrl2 := ctrl.GetICoreWebView2Controller2(); ctrl2 != nil {
			col := edge.COREWEBVIEW2_COLOR{A: 255, R: 15, G: 20, B: 29}
			if !dark {
				col = edge.COREWEBVIEW2_COLOR{A: 255, R: 241, G: 245, B: 249}
			}
			_ = ctrl2.PutDefaultBackgroundColor(col)
		}
	}
}

func (h *winHost) OpenVideoDialog(initialDir string) (string, bool) {
	return OpenVideoDialog(h.hwnd, initialDir)
}

func (h *winHost) OpenExecutableDialog(title, initialDir string) (string, bool) {
	return OpenExeDialog(h.hwnd, title, initialDir)
}

func (h *winHost) SaveVideoDialog(title, initialDir, defaultName, ext string, promptOverwrite bool) (string, bool) {
	return SaveVideoDialog(h.hwnd, title, initialDir, defaultName, ext, promptOverwrite)
}

func (h *winHost) ConfirmOverwriteDialog(title, message string) bool {
	return confirmOverwriteDialog(h.hwnd, title, message)
}

func (h *winHost) RevealInFileManager(path string) {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	hideSubprocessWindow(cmd)
	_ = cmd.Start()
}

// Keep unused windows import clean
var _ = windows.ERROR_SUCCESS
