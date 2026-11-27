//go:build darwin

package main

import (
	_ "embed"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

//go:embed split.png
var splitIconPNGDarwin []byte

type nsPoint struct {
	X, Y float64
}

type nsSize struct {
	Width, Height float64
}

type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

type darwinHost struct {
	nsApp   objc.ID
	window  objc.ID
	webView objc.ID
	app     *splitApp

	mainQ          uintptr
	dispatchAsyncF func(queue, ctx, work uintptr)
	drainCb        uintptr
	dispMu         sync.Mutex
	dispatchQ      []func()

	// Cached selectors & classes
	clsNSString        objc.Class
	clsNSData          objc.Class
	clsNSImage         objc.Class
	clsNSURL           objc.Class
	clsNSURLRequest    objc.Class
	clsNSAppearance    objc.Class
	clsNSOpenPanel     objc.Class
	clsNSSavePanel     objc.Class
	clsNSAlert         objc.Class
	clsNSAutorelease   objc.Class
	selAlloc           objc.SEL
	selInit            objc.SEL
	selRelease         objc.SEL
	selDrain           objc.SEL
	selStringWithUTF8  objc.SEL
	selUTF8String      objc.SEL
	selSetTitle        objc.SEL
	selSetAppearance   objc.SEL
	selAppearanceNamed objc.SEL
	selEvalJS          objc.SEL
	selRunModal        objc.SEL
	selURL             objc.SEL
	selPath            objc.SEL
	selFileURLWithPath objc.SEL
	selSetDirectoryURL objc.SEL
	selSetNameField    objc.SEL
	selSetMessage      objc.SEL
	selSetInfoText     objc.SEL
	selAddButtonTitle  objc.SEL
	selSetAlertStyle   objc.SEL
	selSetAllowsMulti  objc.SEL
	selSetCanDir       objc.SEL
	selSetCanFiles     objc.SEL
	selSetCanCreateDir objc.SEL
}

var (
	activeDarwinHost *darwinHost
	darwinClassesOnce sync.Once
	darwinDelegateCls objc.Class
	darwinWebViewCls  objc.Class
	darwinStrlen      func(uintptr) uintptr
)

func initDarwinRuntime() (mainQ uintptr, dispatchAsyncF func(uintptr, uintptr, uintptr), err error) {
	libSys, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to load libSystem: %w", err)
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return 0, nil, fmt.Errorf("failed to load Foundation.framework: %w", err)
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return 0, nil, fmt.Errorf("failed to load AppKit.framework: %w", err)
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/WebKit.framework/WebKit", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return 0, nil, fmt.Errorf("failed to load WebKit.framework: %w", err)
	}

	mq, err := purego.Dlsym(libSys, "_dispatch_main_q")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to resolve _dispatch_main_q: %w", err)
	}
	purego.RegisterLibFunc(&dispatchAsyncF, libSys, "dispatch_async_f")
	purego.RegisterLibFunc(&darwinStrlen, libSys, "strlen")
	return mq, dispatchAsyncF, nil
}

func (h *darwinHost) initSelectors() {
	h.clsNSString = objc.GetClass("NSString")
	h.clsNSData = objc.GetClass("NSData")
	h.clsNSImage = objc.GetClass("NSImage")
	h.clsNSURL = objc.GetClass("NSURL")
	h.clsNSURLRequest = objc.GetClass("NSURLRequest")
	h.clsNSAppearance = objc.GetClass("NSAppearance")
	h.clsNSOpenPanel = objc.GetClass("NSOpenPanel")
	h.clsNSSavePanel = objc.GetClass("NSSavePanel")
	h.clsNSAlert = objc.GetClass("NSAlert")
	h.clsNSAutorelease = objc.GetClass("NSAutoreleasePool")

	h.selAlloc = objc.RegisterName("alloc")
	h.selInit = objc.RegisterName("init")
	h.selRelease = objc.RegisterName("release")
	h.selDrain = objc.RegisterName("drain")
	h.selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	h.selUTF8String = objc.RegisterName("UTF8String")
	h.selSetTitle = objc.RegisterName("setTitle:")
	h.selSetAppearance = objc.RegisterName("setAppearance:")
	h.selAppearanceNamed = objc.RegisterName("appearanceNamed:")
	h.selEvalJS = objc.RegisterName("evaluateJavaScript:completionHandler:")
	h.selRunModal = objc.RegisterName("runModal")
	h.selURL = objc.RegisterName("URL")
	h.selPath = objc.RegisterName("path")
	h.selFileURLWithPath = objc.RegisterName("fileURLWithPath:")
	h.selSetDirectoryURL = objc.RegisterName("setDirectoryURL:")
	h.selSetNameField = objc.RegisterName("setNameFieldStringValue:")
	h.selSetMessage = objc.RegisterName("setMessageText:")
	h.selSetInfoText = objc.RegisterName("setInformativeText:")
	h.selAddButtonTitle = objc.RegisterName("addButtonWithTitle:")
	h.selSetAlertStyle = objc.RegisterName("setAlertStyle:")
	h.selSetAllowsMulti = objc.RegisterName("setAllowsMultipleSelection:")
	h.selSetCanDir = objc.RegisterName("setCanChooseDirectories:")
	h.selSetCanFiles = objc.RegisterName("setCanChooseFiles:")
	h.selSetCanCreateDir = objc.RegisterName("setCanCreateDirectories:")
}

func (h *darwinHost) nsStr(s string) objc.ID {
	return objc.ID(h.clsNSString).Send(h.selStringWithUTF8, s)
}

func (h *darwinHost) nsStringToGo(nsStr objc.ID) string {
	if nsStr == 0 {
		return ""
	}
	cPtr := uintptr(nsStr.Send(h.selUTF8String))
	if cPtr == 0 {
		return ""
	}
	n := darwinStrlen(cPtr)
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(cPtr)), n))
}

func registerDarwinObjCClasses() {
	darwinClassesOnce.Do(func() {
		nsObjCls := objc.GetClass("NSObject")
		selTerm := objc.RegisterName("applicationShouldTerminateAfterLastWindowClosed:")
		selOpenFile := objc.RegisterName("application:openFile:")
		selWinClose := objc.RegisterName("windowWillClose:")

		var err error
		darwinDelegateCls, err = objc.RegisterClass(
			"SplitDarwinAppDelegate",
			nsObjCls,
			nil,
			nil,
			[]objc.MethodDef{
				{
					Cmd: selTerm,
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
						return true
					},
				},
				{
					Cmd: selOpenFile,
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, filename objc.ID) bool {
						h := activeDarwinHost
						if h == nil || h.app == nil || filename == 0 {
							return false
						}
						path := h.nsStringToGo(filename)
						if path == "" {
							return false
						}
						h.app.loadFileAsync(path)
						return true
					},
				},
				{
					Cmd: selWinClose,
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
						if activeDarwinHost != nil && activeDarwinHost.app != nil {
							activeDarwinHost.app.cancelLoadAndExport()
						}
					},
				},
			},
		)
		if err != nil {
			panic(err)
		}

		wkCls := objc.GetClass("WKWebView")
		selPerformDrag := objc.RegisterName("performDragOperation:")
		selDraggingPboard := objc.RegisterName("draggingPasteboard")
		selURLFromPboard := objc.RegisterName("URLFromPasteboard:")
		selIsFileURL := objc.RegisterName("isFileURL")

		darwinWebViewCls, err = objc.RegisterClass(
			"SplitWKWebView",
			wkCls,
			nil,
			nil,
			[]objc.MethodDef{
				{
					Cmd: selPerformDrag,
					Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) bool {
						h := activeDarwinHost
						if h != nil && h.app != nil && sender != 0 {
							pboard := sender.Send(selDraggingPboard)
							if pboard != 0 {
								nsURL := objc.ID(h.clsNSURL).Send(selURLFromPboard, pboard)
								if nsURL != 0 && objc.Send[bool](nsURL, selIsFileURL) {
									path := h.nsStringToGo(nsURL.Send(h.selPath))
									if path != "" {
										h.EvalJS(`
											document.getElementById("dropCard")?.classList.remove("drag-over");
											document.getElementById("globalDropOverlay")?.classList.add("hidden");
										`)
										h.app.loadFileAsync(path)
										return true
									}
								}
							}
						}
						return objc.SendSuper[bool](self, selPerformDrag, sender)
					},
				},
			},
		)
		if err != nil {
			panic(err)
		}
	})
}

func darwinDrainQueue(_ uintptr) {
	h := activeDarwinHost
	if h == nil {
		return
	}
	pool := objc.ID(h.clsNSAutorelease).Send(h.selAlloc).Send(h.selInit)
	defer pool.Send(h.selDrain)

	h.dispMu.Lock()
	q := append([]func(){}, h.dispatchQ...)
	h.dispatchQ = nil
	h.dispMu.Unlock()

	for _, fn := range q {
		fn()
	}
}

func runGUI(preloadPath string) error {
	runtime.LockOSThread()

	mainQ, asyncFn, err := initDarwinRuntime()
	if err != nil {
		return err
	}

	settings := NewSettingsStore()
	cfg := settings.Get()
	tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)

	srv, err := startMediaServer(tools.FFmpegPath)
	if err != nil {
		return fmt.Errorf("failed to start local media server: %w", err)
	}
	defer srv.Close()

	host := &darwinHost{
		mainQ:          mainQ,
		dispatchAsyncF: asyncFn,
		drainCb:        purego.NewCallback(darwinDrainQueue),
	}
	host.initSelectors()
	registerDarwinObjCClasses()

	app := newSplitApp(srv, settings, tools, host)
	host.app = app
	activeDarwinHost = host

	pool := objc.ID(host.clsNSAutorelease).Send(host.selAlloc).Send(host.selInit)

	nsAppCls := objc.GetClass("NSApplication")
	nsApp := objc.ID(nsAppCls).Send(objc.RegisterName("sharedApplication"))
	host.nsApp = nsApp
	// NSApplicationActivationPolicyRegular = 0
	nsApp.Send(objc.RegisterName("setActivationPolicy:"), uintptr(0))

	host.installAppIcon()
	host.installMainMenu()

	delegate := objc.ID(darwinDelegateCls).Send(host.selAlloc).Send(host.selInit)
	nsApp.Send(objc.RegisterName("setDelegate:"), delegate)

	// NSWindowStyleMask: Titled(1) | Closable(2) | Miniaturizable(4) | Resizable(8) = 15
	rect := nsRect{Origin: nsPoint{X: 120, Y: 120}, Size: nsSize{Width: 1240, Height: 800}}
	winCls := objc.GetClass("NSWindow")
	win := objc.ID(winCls).Send(host.selAlloc).Send(
		objc.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect,
		uintptr(15),
		uintptr(2), // NSBackingStoreBuffered
		false,
	)
	host.window = win
	win.Send(host.selSetTitle, host.nsStr("Split"))
	win.Send(objc.RegisterName("setMinSize:"), nsSize{Width: 940, Height: 620})
	win.Send(objc.RegisterName("setDelegate:"), delegate)
	win.Send(objc.RegisterName("center"))

	// WKWebViewConfiguration with inline playback allowed
	wkCfgCls := objc.GetClass("WKWebViewConfiguration")
	wkCfg := objc.ID(wkCfgCls).Send(host.selAlloc).Send(host.selInit)
	wkCfg.Send(objc.RegisterName("setMediaTypesRequiringUserActionForPlayback:"), uintptr(0))

	contentView := win.Send(objc.RegisterName("contentView"))
	bounds := nsRect{Origin: nsPoint{X: 0, Y: 0}, Size: nsSize{Width: 1240, Height: 800}}
	wv := objc.ID(darwinWebViewCls).Send(host.selAlloc).Send(
		objc.RegisterName("initWithFrame:configuration:"),
		bounds,
		wkCfg,
	)
	host.webView = wv
	// NSViewWidthSizable(2) | NSViewHeightSizable(16) = 18
	wv.Send(objc.RegisterName("setAutoresizingMask:"), uintptr(18))
	// Register file URL drag type explicitly on WebView
	nsArrayCls := objc.GetClass("NSArray")
	dragTypes := objc.ID(nsArrayCls).Send(objc.RegisterName("arrayWithObject:"), host.nsStr("public.file-url"))
	wv.Send(objc.RegisterName("registerForDraggedTypes:"), dragTypes)

	contentView.Send(objc.RegisterName("addSubview:"), wv)

	host.ApplyTheme(cfg.Theme)

	urlObj := objc.ID(host.clsNSURL).Send(objc.RegisterName("URLWithString:"), host.nsStr(srv.IndexURL()))
	reqObj := objc.ID(host.clsNSURLRequest).Send(objc.RegisterName("requestWithURL:"), urlObj)
	wv.Send(objc.RegisterName("loadRequest:"), reqObj)

	win.Send(objc.RegisterName("makeKeyAndOrderFront:"), objc.ID(0))
	nsApp.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)

	pool.Send(host.selDrain)

	if preloadPath != "" {
		app.loadFileAsync(preloadPath)
	}

	nsApp.Send(objc.RegisterName("run"))
	return nil
}

func (h *darwinHost) installAppIcon() {
	if len(splitIconPNGDarwin) == 0 || h.clsNSData == 0 || h.clsNSImage == 0 {
		return
	}
	data := objc.ID(h.clsNSData).Send(
		objc.RegisterName("dataWithBytes:length:"),
		uintptr(unsafe.Pointer(&splitIconPNGDarwin[0])),
		uintptr(len(splitIconPNGDarwin)),
	)
	runtime.KeepAlive(splitIconPNGDarwin)
	if data == 0 {
		return
	}
	img := objc.ID(h.clsNSImage).Send(h.selAlloc).Send(objc.RegisterName("initWithData:"), data)
	if img != 0 {
		h.nsApp.Send(objc.RegisterName("setApplicationIconImage:"), img)
		img.Send(h.selRelease)
	}
}

func (h *darwinHost) installMainMenu() {
	nsMenuCls := objc.GetClass("NSMenu")
	nsMenuItemCls := objc.GetClass("NSMenuItem")
	selAddItem := objc.RegisterName("addItem:")
	selSetSubmenu := objc.RegisterName("setSubmenu:")
	selInitTitleActionKey := objc.RegisterName("initWithTitle:action:keyEquivalent:")

	mainMenu := objc.ID(nsMenuCls).Send(h.selAlloc).Send(h.selInit)
	appMenuItem := objc.ID(nsMenuItemCls).Send(h.selAlloc).Send(h.selInit)
	mainMenu.Send(selAddItem, appMenuItem)

	appMenu := objc.ID(nsMenuCls).Send(h.selAlloc).Send(h.selInit)
	quitItem := objc.ID(nsMenuItemCls).Send(h.selAlloc).Send(
		selInitTitleActionKey,
		h.nsStr("Quit Split"),
		objc.RegisterName("terminate:"),
		h.nsStr("q"),
	)
	appMenu.Send(selAddItem, quitItem)
	appMenuItem.Send(selSetSubmenu, appMenu)
	h.nsApp.Send(objc.RegisterName("setMainMenu:"), mainMenu)
}

func (h *darwinHost) DispatchUI(fn func()) {
	h.dispMu.Lock()
	h.dispatchQ = append(h.dispatchQ, fn)
	h.dispMu.Unlock()
	h.dispatchAsyncF(h.mainQ, 0, h.drainCb)
}

func (h *darwinHost) EvalJS(script string) {
	if h.webView == 0 {
		return
	}
	h.DispatchUI(func() {
		h.webView.Send(h.selEvalJS, h.nsStr(script), uintptr(0))
	})
}

func (h *darwinHost) SetWindowTitle(title string) {
	if h.window == 0 {
		return
	}
	h.DispatchUI(func() {
		h.window.Send(h.selSetTitle, h.nsStr(title))
	})
}

func (h *darwinHost) ApplyTheme(theme string) {
	if h.nsApp == 0 {
		return
	}
	name := "NSAppearanceNameDarkAqua"
	if theme == "light" {
		name = "NSAppearanceNameAqua"
	}
	app := objc.ID(h.clsNSAppearance).Send(h.selAppearanceNamed, h.nsStr(name))
	if app != 0 {
		h.nsApp.Send(h.selSetAppearance, app)
		if h.window != 0 {
			h.window.Send(h.selSetAppearance, app)
		}
	}
}

func (h *darwinHost) OpenVideoDialog(initialDir string) (string, bool) {
	panel := objc.ID(h.clsNSOpenPanel).Send(objc.RegisterName("openPanel"))
	panel.Send(h.selSetTitle, h.nsStr("Open Video File"))
	panel.Send(h.selSetAllowsMulti, false)
	panel.Send(h.selSetCanDir, false)
	panel.Send(h.selSetCanFiles, true)
	if initialDir != "" {
		dirURL := objc.ID(h.clsNSURL).Send(h.selFileURLWithPath, h.nsStr(initialDir))
		if dirURL != 0 {
			panel.Send(h.selSetDirectoryURL, dirURL)
		}
	}
	// NSModalResponseOK = 1
	if int(panel.Send(h.selRunModal)) != 1 {
		return "", false
	}
	u := panel.Send(h.selURL)
	if u == 0 {
		return "", false
	}
	p := h.nsStringToGo(u.Send(h.selPath))
	if p == "" {
		return "", false
	}
	return p, true
}

func (h *darwinHost) OpenExecutableDialog(title, initialDir string) (string, bool) {
	panel := objc.ID(h.clsNSOpenPanel).Send(objc.RegisterName("openPanel"))
	panel.Send(h.selSetTitle, h.nsStr(title))
	panel.Send(h.selSetAllowsMulti, false)
	panel.Send(h.selSetCanDir, false)
	panel.Send(h.selSetCanFiles, true)
	if initialDir != "" {
		dirURL := objc.ID(h.clsNSURL).Send(h.selFileURLWithPath, h.nsStr(initialDir))
		if dirURL != 0 {
			panel.Send(h.selSetDirectoryURL, dirURL)
		}
	}
	if int(panel.Send(h.selRunModal)) != 1 {
		return "", false
	}
	u := panel.Send(h.selURL)
	if u == 0 {
		return "", false
	}
	p := h.nsStringToGo(u.Send(h.selPath))
	if p == "" {
		return "", false
	}
	return p, true
}

func (h *darwinHost) SaveVideoDialog(title, initialDir, defaultName, ext string, _ bool) (string, bool) {
	panel := objc.ID(h.clsNSSavePanel).Send(objc.RegisterName("savePanel"))
	panel.Send(h.selSetTitle, h.nsStr(title))
	panel.Send(h.selSetCanCreateDir, true)
	if defaultName != "" {
		panel.Send(h.selSetNameField, h.nsStr(defaultName))
	}
	if initialDir != "" {
		dirURL := objc.ID(h.clsNSURL).Send(h.selFileURLWithPath, h.nsStr(initialDir))
		if dirURL != 0 {
			panel.Send(h.selSetDirectoryURL, dirURL)
		}
	}
	if int(panel.Send(h.selRunModal)) != 1 {
		return "", false
	}
	u := panel.Send(h.selURL)
	if u == 0 {
		return "", false
	}
	chosen := h.nsStringToGo(u.Send(h.selPath))
	if chosen == "" {
		return "", false
	}
	cleanExt := strings.TrimPrefix(strings.ToLower(ext), ".")
	if filepath.Ext(chosen) == "" && cleanExt != "" {
		chosen += "." + cleanExt
	}
	return chosen, true
}

func (h *darwinHost) ConfirmOverwriteDialog(title, message string) bool {
	alert := objc.ID(h.clsNSAlert).Send(h.selAlloc).Send(h.selInit)
	alert.Send(h.selSetAlertStyle, uintptr(0)) // NSAlertStyleWarning = 0
	alert.Send(h.selSetMessage, h.nsStr(title))
	alert.Send(h.selSetInfoText, h.nsStr(message))
	alert.Send(h.selAddButtonTitle, h.nsStr("Overwrite"))
	alert.Send(h.selAddButtonTitle, h.nsStr("Cancel"))
	// NSAlertFirstButtonReturn = 1000
	return int(alert.Send(h.selRunModal)) == 1000
}

func (h *darwinHost) RevealInFileManager(path string) {
	_ = exec.Command("open", "-R", path).Start()
}

func showMessageBoxError(title, msg string) bool {
	mainQ, _, err := initDarwinRuntime()
	if err != nil || mainQ == 0 {
		return false
	}
	h := &darwinHost{}
	h.initSelectors()
	pool := objc.ID(h.clsNSAutorelease).Send(h.selAlloc).Send(h.selInit)
	defer pool.Send(h.selDrain)

	nsApp := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	nsApp.Send(objc.RegisterName("setActivationPolicy:"), uintptr(0))

	alert := objc.ID(h.clsNSAlert).Send(h.selAlloc).Send(h.selInit)
	alert.Send(h.selSetAlertStyle, uintptr(2)) // NSAlertStyleCritical = 2
	alert.Send(h.selSetMessage, h.nsStr(title))
	alert.Send(h.selSetInfoText, h.nsStr(msg))
	alert.Send(h.selAddButtonTitle, h.nsStr("OK"))
	alert.Send(h.selRunModal)
	return true
}
