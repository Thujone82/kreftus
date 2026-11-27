//go:build linux

package main

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

//go:embed split.png
var splitIconPNGLinux []byte

type gValue struct {
	GType uintptr
	Data  [2]uint64
}

type linuxHost struct {
	window  uintptr
	webView uintptr
	app     *splitApp

	dispMu    sync.Mutex
	dispatchQ []func()
	idleCb    uintptr

	hasEvalJS bool
}

var (
	activeLinuxHost *linuxHost

	gtkInitOnce sync.Once
	gtkInitErr  error

	// GLib / GObject
	gIdleAdd           func(fn, data uintptr) uint32
	gFree              func(ptr uintptr)
	gStrfreev          func(strv uintptr)
	gObjectUnref       func(obj uintptr)
	gSignalConnectData func(instance uintptr, signal string, handler, data, destroyNotify uintptr, flags int32) uint64
	gObjectSetProperty func(obj uintptr, name string, val uintptr)
	gValueInit         func(val, gType uintptr) uintptr
	gValueSetBoolean   func(val uintptr, v int32)
	gValueUnset        func(val uintptr)
	linuxStrlen        func(uintptr) uintptr

	// GdkPixbuf (optional icon)
	gdkPixbufLoaderNew       func() uintptr
	gdkPixbufLoaderWrite     func(loader uintptr, buf *byte, count uintptr, errOut uintptr) int32
	gdkPixbufLoaderClose     func(loader uintptr, errOut uintptr) int32
	gdkPixbufLoaderGetPixbuf func(loader uintptr) uintptr

	// GTK3
	gtkInitCheck                    func(argc, argv uintptr) int32
	gtkMain                         func()
	gtkMainQuit                     func()
	gtkWindowNew                    func(winType int32) uintptr
	gtkWindowSetTitle               func(win uintptr, title string)
	gtkWindowSetDefaultSize         func(win uintptr, width, height int32)
	gtkWindowSetPosition            func(win uintptr, pos int32)
	gtkWindowSetIcon                func(win uintptr, pixbuf uintptr)
	gtkWindowSetTransientFor        func(win, parent uintptr)
	gtkWindowSetModal               func(win uintptr, modal int32)
	gtkWidgetSetSizeRequest         func(w uintptr, width, height int32)
	gtkWidgetShowAll                func(w uintptr)
	gtkWidgetDestroy                func(w uintptr)
	gtkContainerAdd                 func(container, widget uintptr)
	gtkContainerSetBorderWidth      func(container uintptr, width uint32)
	gtkSettingsGetDefault           func() uintptr
	gtkDragDestSet                  func(widget uintptr, flags int32, targets uintptr, nTargets, actions int32)
	gtkDragDestAddURITargets        func(widget uintptr)
	gtkDragFinish                   func(ctx uintptr, success, del int32, t uint32)
	gtkSelectionDataGetURIs         func(data uintptr) uintptr
	gtkFileChooserNativeNew         func(title string, parent uintptr, action int32, acceptLabel, cancelLabel string) uintptr
	gtkNativeDialogRun              func(dlg uintptr) int32
	gtkFileChooserSetCurrentFolder  func(chooser uintptr, folder string) int32
	gtkFileChooserSetCurrentName    func(chooser uintptr, name string)
	gtkFileChooserSetDoOverwrite    func(chooser uintptr, doOverwrite int32)
	gtkFileChooserGetFilename       func(chooser uintptr) uintptr
	gtkFileChooserAddFilter         func(chooser, filter uintptr)
	gtkFileFilterNew                func() uintptr
	gtkFileFilterSetName            func(filter uintptr, name string)
	gtkFileFilterAddPattern         func(filter uintptr, pattern string)
	gtkDialogNew                    func() uintptr
	gtkDialogGetContentArea         func(dlg uintptr) uintptr
	gtkDialogAddButton              func(dlg uintptr, text string, respID int32) uintptr
	gtkDialogRun                    func(dlg uintptr) int32
	gtkLabelNew                     func(text string) uintptr
	gtkLabelSetLineWrap             func(lbl uintptr, wrap int32)

	// WebKit2GTK
	webkitWebViewNew         func() uintptr
	webkitWebViewLoadURI     func(wv uintptr, uri string)
	webkitWebViewEvalJS      func(wv uintptr, script string, length int, worldName, sourceURI, cancellable, callback, userData uintptr)
	webkitWebViewRunJS       func(wv uintptr, script string, cancellable, callback, userData uintptr)
)

func dlopenFirst(names []string) (uintptr, error) {
	var lastErr error
	for _, n := range names {
		h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err == nil && h != 0 {
			return h, nil
		}
		lastErr = err
	}
	return 0, lastErr
}

func bindSym(fptr any, lib uintptr, name string) error {
	sym, err := purego.Dlsym(lib, name)
	if err != nil {
		return fmt.Errorf("symbol %s: %w", name, err)
	}
	purego.RegisterFunc(fptr, sym)
	return nil
}

func initLinuxLibs() error {
	gtkInitOnce.Do(func() {
		libc, err := dlopenFirst([]string{"libc.so.6", "libc.so"})
		if err == nil {
			_ = bindSym(&linuxStrlen, libc, "strlen")
		}

		glib, err := dlopenFirst([]string{"libglib-2.0.so.0", "libglib-2.0.so"})
		if err != nil {
			gtkInitErr = fmt.Errorf("missing GLib (libglib-2.0.so.0): %w\n\nInstall on Debian/Ubuntu:\nsudo apt install libgtk-3-0 libwebkit2gtk-4.1-0", err)
			return
		}
		gobj, err := dlopenFirst([]string{"libgobject-2.0.so.0", "libgobject-2.0.so"})
		if err != nil {
			gtkInitErr = fmt.Errorf("missing GObject (libgobject-2.0.so.0): %w", err)
			return
		}
		gtk, err := dlopenFirst([]string{"libgtk-3.so.0", "libgtk-3.so"})
		if err != nil {
			gtkInitErr = fmt.Errorf("missing GTK3 (libgtk-3.so.0): %w\n\nInstall on Debian/Ubuntu:\nsudo apt install libgtk-3-0 libwebkit2gtk-4.1-0", err)
			return
		}
		wk, err := dlopenFirst([]string{"libwebkit2gtk-4.1.so.0", "libwebkit2gtk-4.0.so.37", "libwebkit2gtk-4.1.so"})
		if err != nil {
			gtkInitErr = fmt.Errorf("missing WebKit2GTK (libwebkit2gtk-4.1.so.0): %w\n\nInstall on Debian/Ubuntu:\nsudo apt install libwebkit2gtk-4.1-0 gstreamer1.0-libav", err)
			return
		}

		if pixbuf, pErr := dlopenFirst([]string{"libgdk_pixbuf-2.0.so.0", "libgdk_pixbuf-2.0.so"}); pErr == nil {
			_ = bindSym(&gdkPixbufLoaderNew, pixbuf, "gdk_pixbuf_loader_new")
			_ = bindSym(&gdkPixbufLoaderWrite, pixbuf, "gdk_pixbuf_loader_write")
			_ = bindSym(&gdkPixbufLoaderClose, pixbuf, "gdk_pixbuf_loader_close")
			_ = bindSym(&gdkPixbufLoaderGetPixbuf, pixbuf, "gdk_pixbuf_loader_get_pixbuf")
		}

		req := []struct {
			fn   any
			lib  uintptr
			name string
		}{
			{&gIdleAdd, glib, "g_idle_add"},
			{&gFree, glib, "g_free"},
			{&gStrfreev, glib, "g_strfreev"},
			{&gObjectUnref, gobj, "g_object_unref"},
			{&gSignalConnectData, gobj, "g_signal_connect_data"},
			{&gObjectSetProperty, gobj, "g_object_set_property"},
			{&gValueInit, gobj, "g_value_init"},
			{&gValueSetBoolean, gobj, "g_value_set_boolean"},
			{&gValueUnset, gobj, "g_value_unset"},

			{&gtkInitCheck, gtk, "gtk_init_check"},
			{&gtkMain, gtk, "gtk_main"},
			{&gtkMainQuit, gtk, "gtk_main_quit"},
			{&gtkWindowNew, gtk, "gtk_window_new"},
			{&gtkWindowSetTitle, gtk, "gtk_window_set_title"},
			{&gtkWindowSetDefaultSize, gtk, "gtk_window_set_default_size"},
			{&gtkWindowSetPosition, gtk, "gtk_window_set_position"},
			{&gtkWindowSetIcon, gtk, "gtk_window_set_icon"},
			{&gtkWindowSetTransientFor, gtk, "gtk_window_set_transient_for"},
			{&gtkWindowSetModal, gtk, "gtk_window_set_modal"},
			{&gtkWidgetSetSizeRequest, gtk, "gtk_widget_set_size_request"},
			{&gtkWidgetShowAll, gtk, "gtk_widget_show_all"},
			{&gtkWidgetDestroy, gtk, "gtk_widget_destroy"},
			{&gtkContainerAdd, gtk, "gtk_container_add"},
			{&gtkContainerSetBorderWidth, gtk, "gtk_container_set_border_width"},
			{&gtkSettingsGetDefault, gtk, "gtk_settings_get_default"},
			{&gtkDragDestSet, gtk, "gtk_drag_dest_set"},
			{&gtkDragDestAddURITargets, gtk, "gtk_drag_dest_add_uri_targets"},
			{&gtkDragFinish, gtk, "gtk_drag_finish"},
			{&gtkSelectionDataGetURIs, gtk, "gtk_selection_data_get_uris"},
			{&gtkFileChooserNativeNew, gtk, "gtk_file_chooser_native_new"},
			{&gtkNativeDialogRun, gtk, "gtk_native_dialog_run"},
			{&gtkFileChooserSetCurrentFolder, gtk, "gtk_file_chooser_set_current_folder"},
			{&gtkFileChooserSetCurrentName, gtk, "gtk_file_chooser_set_current_name"},
			{&gtkFileChooserSetDoOverwrite, gtk, "gtk_file_chooser_set_do_overwrite_confirmation"},
			{&gtkFileChooserGetFilename, gtk, "gtk_file_chooser_get_filename"},
			{&gtkFileChooserAddFilter, gtk, "gtk_file_chooser_add_filter"},
			{&gtkFileFilterNew, gtk, "gtk_file_filter_new"},
			{&gtkFileFilterSetName, gtk, "gtk_file_filter_set_name"},
			{&gtkFileFilterAddPattern, gtk, "gtk_file_filter_add_pattern"},
			{&gtkDialogNew, gtk, "gtk_dialog_new"},
			{&gtkDialogGetContentArea, gtk, "gtk_dialog_get_content_area"},
			{&gtkDialogAddButton, gtk, "gtk_dialog_add_button"},
			{&gtkDialogRun, gtk, "gtk_dialog_run"},
			{&gtkLabelNew, gtk, "gtk_label_new"},
			{&gtkLabelSetLineWrap, gtk, "gtk_label_set_line_wrap"},

			{&webkitWebViewNew, wk, "webkit_web_view_new"},
			{&webkitWebViewLoadURI, wk, "webkit_web_view_load_uri"},
		}
		for _, r := range req {
			if err := bindSym(r.fn, r.lib, r.name); err != nil {
				gtkInitErr = err
				return
			}
		}

		if bindSym(&webkitWebViewEvalJS, wk, "webkit_web_view_evaluate_javascript") != nil {
			if err := bindSym(&webkitWebViewRunJS, wk, "webkit_web_view_run_javascript"); err != nil {
				gtkInitErr = err
				return
			}
		}

		if gtkInitCheck(0, 0) == 0 {
			gtkInitErr = fmt.Errorf("failed to initialize GTK display (ensure DISPLAY or WAYLAND_DISPLAY is set)")
			return
		}
	})
	return gtkInitErr
}

func cStrToGo(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var n uintptr
	if linuxStrlen != nil {
		n = linuxStrlen(ptr)
	} else {
		for *(*byte)(unsafe.Pointer(ptr + n)) != 0 {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
}

func linuxDrainIdle(_ uintptr) int32 {
	h := activeLinuxHost
	if h == nil {
		return 0
	}
	h.dispMu.Lock()
	q := append([]func(){}, h.dispatchQ...)
	h.dispatchQ = nil
	h.dispMu.Unlock()

	for _, fn := range q {
		fn()
	}
	return 0 // G_SOURCE_REMOVE
}

func linuxOnDestroy(_, _ uintptr) {
	if activeLinuxHost != nil && activeLinuxHost.app != nil {
		activeLinuxHost.app.cancelLoadAndExport()
	}
	gtkMainQuit()
}

func linuxOnDragDataReceived(_, dragCtx uintptr, _, _ int32, selData uintptr, _, timeStamp uint32, _ uintptr) {
	h := activeLinuxHost
	if h == nil || h.app == nil || selData == 0 {
		return
	}
	urisPtr := gtkSelectionDataGetURIs(selData)
	if urisPtr == 0 {
		return
	}
	defer gStrfreev(urisPtr)

	ptrSize := unsafe.Sizeof(uintptr(0))
	var loaded bool
	for idx := uintptr(0); ; idx++ {
		elemPtr := *(*uintptr)(unsafe.Pointer(urisPtr + idx*ptrSize))
		if elemPtr == 0 {
			break
		}
		rawURI := cStrToGo(elemPtr)
		u, err := url.Parse(rawURI)
		if err != nil || u.Scheme != "file" {
			continue
		}
		localPath := u.Path
		if st, sErr := os.Stat(localPath); sErr == nil && !st.IsDir() {
			h.EvalJS(`
				document.getElementById("dropCard")?.classList.remove("drag-over");
				document.getElementById("globalDropOverlay")?.classList.add("hidden");
			`)
			h.app.loadFileAsync(localPath)
			loaded = true
			break
		}
	}
	if dragCtx != 0 {
		okInt := int32(0)
		if loaded {
			okInt = 1
		}
		gtkDragFinish(dragCtx, okInt, 0, timeStamp)
	}
}

func runGUI(preloadPath string) error {
	runtime.LockOSThread()

	if err := initLinuxLibs(); err != nil {
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

	host := &linuxHost{
		idleCb:    purego.NewCallback(linuxDrainIdle),
		hasEvalJS: webkitWebViewEvalJS != nil,
	}
	app := newSplitApp(srv, settings, tools, host)
	host.app = app
	activeLinuxHost = host

	win := gtkWindowNew(0) // GTK_WINDOW_TOPLEVEL
	host.window = win
	gtkWindowSetTitle(win, "Split")
	gtkWindowSetDefaultSize(win, 1240, 800)
	gtkWindowSetPosition(win, 1) // GTK_WIN_POS_CENTER
	gtkWidgetSetSizeRequest(win, 940, 620)
	host.installWindowIcon()

	wv := webkitWebViewNew()
	host.webView = wv
	gtkContainerAdd(win, wv)

	destroyCb := purego.NewCallback(linuxOnDestroy)
	gSignalConnectData(win, "destroy", destroyCb, 0, 0, 0)

	// Enable URI drop on both window and WebView so OS file manager drops pass native paths cleanly.
	gtkDragDestSet(win, 7 /* GTK_DEST_DEFAULT_ALL */, 0, 0, 1|2|4 /* COPY|MOVE|LINK */)
	gtkDragDestAddURITargets(win)
	dragCb := purego.NewCallback(linuxOnDragDataReceived)
	gSignalConnectData(win, "drag-data-received", dragCb, 0, 0, 0)
	gSignalConnectData(wv, "drag-data-received", dragCb, 0, 0, 0)

	host.ApplyTheme(cfg.Theme)
	webkitWebViewLoadURI(wv, srv.IndexURL())
	gtkWidgetShowAll(win)

	if preloadPath != "" {
		app.loadFileAsync(preloadPath)
	}

	gtkMain()
	return nil
}

func (h *linuxHost) installWindowIcon() {
	if h.window == 0 || len(splitIconPNGLinux) == 0 || gdkPixbufLoaderNew == nil || gdkPixbufLoaderWrite == nil || gdkPixbufLoaderClose == nil || gdkPixbufLoaderGetPixbuf == nil {
		return
	}
	loader := gdkPixbufLoaderNew()
	if loader == 0 {
		return
	}
	defer gObjectUnref(loader)
	ok := gdkPixbufLoaderWrite(loader, &splitIconPNGLinux[0], uintptr(len(splitIconPNGLinux)), 0)
	runtime.KeepAlive(splitIconPNGLinux)
	_ = gdkPixbufLoaderClose(loader, 0)
	if ok == 0 {
		return
	}
	pb := gdkPixbufLoaderGetPixbuf(loader)
	if pb != 0 {
		gtkWindowSetIcon(h.window, pb)
	}
}

func (h *linuxHost) DispatchUI(fn func()) {
	h.dispMu.Lock()
	h.dispatchQ = append(h.dispatchQ, fn)
	h.dispMu.Unlock()
	gIdleAdd(h.idleCb, 0)
}

func (h *linuxHost) EvalJS(script string) {
	if h.webView == 0 {
		return
	}
	h.DispatchUI(func() {
		if h.hasEvalJS {
			webkitWebViewEvalJS(h.webView, script, -1, 0, 0, 0, 0, 0)
		} else if webkitWebViewRunJS != nil {
			webkitWebViewRunJS(h.webView, script, 0, 0, 0)
		}
	})
}

func (h *linuxHost) SetWindowTitle(title string) {
	if h.window == 0 {
		return
	}
	h.DispatchUI(func() {
		gtkWindowSetTitle(h.window, title)
	})
}

func (h *linuxHost) ApplyTheme(theme string) {
	settings := gtkSettingsGetDefault()
	if settings == 0 {
		return
	}
	var gv gValue
	// G_TYPE_BOOLEAN = 5 << 2 = 20
	gValueInit(uintptr(unsafe.Pointer(&gv)), 20)
	b := int32(1)
	if theme == "light" {
		b = 0
	}
	gValueSetBoolean(uintptr(unsafe.Pointer(&gv)), b)
	gObjectSetProperty(settings, "gtk-application-prefer-dark-theme", uintptr(unsafe.Pointer(&gv)))
	gValueUnset(uintptr(unsafe.Pointer(&gv)))
}

func (h *linuxHost) OpenVideoDialog(initialDir string) (string, bool) {
	dlg := gtkFileChooserNativeNew("Open Video File", h.window, 0 /* OPEN */, "Open", "Cancel")
	if dlg == 0 {
		return "", false
	}
	defer gObjectUnref(dlg)

	fVid := gtkFileFilterNew()
	gtkFileFilterSetName(fVid, "Video Files")
	for _, pat := range []string{"*.mp4", "*.mkv", "*.mov", "*.webm", "*.avi", "*.ts", "*.m4v", "*.flv", "*.wmv", "*.MP4", "*.MKV", "*.MOV", "*.TS"} {
		gtkFileFilterAddPattern(fVid, pat)
	}
	gtkFileChooserAddFilter(dlg, fVid)

	fAll := gtkFileFilterNew()
	gtkFileFilterSetName(fAll, "All Files")
	gtkFileFilterAddPattern(fAll, "*")
	gtkFileChooserAddFilter(dlg, fAll)

	if initialDir != "" {
		gtkFileChooserSetCurrentFolder(dlg, initialDir)
	}
	if gtkNativeDialogRun(dlg) != -3 /* GTK_RESPONSE_ACCEPT */ {
		return "", false
	}
	cPath := gtkFileChooserGetFilename(dlg)
	if cPath == 0 {
		return "", false
	}
	defer gFree(cPath)
	p := cStrToGo(cPath)
	if p == "" {
		return "", false
	}
	return p, true
}

func (h *linuxHost) OpenExecutableDialog(title, initialDir string) (string, bool) {
	dlg := gtkFileChooserNativeNew(title, h.window, 0 /* OPEN */, "Select", "Cancel")
	if dlg == 0 {
		return "", false
	}
	defer gObjectUnref(dlg)

	if initialDir != "" {
		gtkFileChooserSetCurrentFolder(dlg, initialDir)
	}
	if gtkNativeDialogRun(dlg) != -3 {
		return "", false
	}
	cPath := gtkFileChooserGetFilename(dlg)
	if cPath == 0 {
		return "", false
	}
	defer gFree(cPath)
	p := cStrToGo(cPath)
	if p == "" {
		return "", false
	}
	return p, true
}

func (h *linuxHost) SaveVideoDialog(title, initialDir, defaultName, ext string, promptOverwrite bool) (string, bool) {
	dlg := gtkFileChooserNativeNew(title, h.window, 1 /* SAVE */, "Save", "Cancel")
	if dlg == 0 {
		return "", false
	}
	defer gObjectUnref(dlg)

	if initialDir != "" {
		gtkFileChooserSetCurrentFolder(dlg, initialDir)
	}
	if defaultName != "" {
		gtkFileChooserSetCurrentName(dlg, defaultName)
	}
	if promptOverwrite {
		gtkFileChooserSetDoOverwrite(dlg, 1)
	}
	if gtkNativeDialogRun(dlg) != -3 {
		return "", false
	}
	cPath := gtkFileChooserGetFilename(dlg)
	if cPath == 0 {
		return "", false
	}
	defer gFree(cPath)
	chosen := cStrToGo(cPath)
	if chosen == "" {
		return "", false
	}
	cleanExt := strings.TrimPrefix(strings.ToLower(ext), ".")
	if filepath.Ext(chosen) == "" && cleanExt != "" {
		chosen += "." + cleanExt
	}
	return chosen, true
}

func (h *linuxHost) ConfirmOverwriteDialog(title, message string) bool {
	dlg := gtkDialogNew()
	if dlg == 0 {
		return false
	}
	gtkWindowSetTitle(dlg, title)
	if h.window != 0 {
		gtkWindowSetTransientFor(dlg, h.window)
	}
	gtkWindowSetModal(dlg, 1)
	area := gtkDialogGetContentArea(dlg)
	gtkContainerSetBorderWidth(area, 14)
	lbl := gtkLabelNew(message)
	gtkLabelSetLineWrap(lbl, 1)
	gtkContainerAdd(area, lbl)
	gtkDialogAddButton(dlg, "Cancel", -6 /* GTK_RESPONSE_CANCEL */)
	gtkDialogAddButton(dlg, "Overwrite", -5 /* GTK_RESPONSE_OK */)
	gtkWidgetShowAll(dlg)
	resp := gtkDialogRun(dlg)
	gtkWidgetDestroy(dlg)
	return resp == -5
}

func (h *linuxHost) RevealInFileManager(path string) {
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	cmd := exec.Command("dbus-send", "--session", "--dest=org.freedesktop.FileManager1",
		"--type=method_call", "/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowItems", "array:string:"+uri, "string:")
	if cmd.Run() == nil {
		return
	}
	_ = exec.Command("xdg-open", filepath.Dir(path)).Start()
}

func showMessageBoxError(title, msg string) bool {
	if initLinuxLibs() != nil {
		return false
	}
	dlg := gtkDialogNew()
	if dlg == 0 {
		return false
	}
	gtkWindowSetTitle(dlg, title)
	gtkWindowSetModal(dlg, 1)
	area := gtkDialogGetContentArea(dlg)
	gtkContainerSetBorderWidth(area, 16)
	lbl := gtkLabelNew(msg)
	gtkLabelSetLineWrap(lbl, 1)
	gtkContainerAdd(area, lbl)
	gtkDialogAddButton(dlg, "OK", -5)
	gtkWidgetShowAll(dlg)
	_ = gtkDialogRun(dlg)
	gtkWidgetDestroy(dlg)
	return true
}
