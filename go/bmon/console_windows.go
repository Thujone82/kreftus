//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// configureWindowsConsole ensures the Windows console uses UTF-8 code page
// and enables Virtual Terminal (ANSI) processing for proper Unicode rendering.
func configureWindowsConsole() {
	// Quietly set UTF-8 code page for legacy console APIs
	_ = exec.Command("cmd", "/c", "chcp 65001 >nul").Run()

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")
	setConsoleCP := kernel32.NewProc("SetConsoleCP")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	const (
		CP_UTF8                         = 65001
		ENABLE_VTP_OUTPUT        uint32 = 0x0004
		ENABLE_PROCESSED_OUTPUT  uint32 = 0x0001
		DISABLE_NEWLINE_AUTO_RET uint32 = 0x0008
		ENABLE_VTP_INPUT         uint32 = 0x0200
	)

	// Set UTF-8 code pages
	_, _, _ = setConsoleOutputCP.Call(uintptr(CP_UTF8))
	_, _, _ = setConsoleCP.Call(uintptr(CP_UTF8))

	// Enable VT on stdout
	outHandle, _, _ := getStdHandle.Call(uintptr(^uint32(11) + 1)) // STD_OUTPUT_HANDLE=-11
	if outHandle != 0 {
		var mode uint32
		_, _, _ = getConsoleMode.Call(outHandle, uintptr(unsafe.Pointer(&mode)))
		mode |= ENABLE_VTP_OUTPUT | ENABLE_PROCESSED_OUTPUT | DISABLE_NEWLINE_AUTO_RET
		_, _, _ = setConsoleMode.Call(outHandle, uintptr(mode))
	}
	// Enable VT on stderr
	errHandle, _, _ := getStdHandle.Call(uintptr(^uint32(12) + 1)) // STD_ERROR_HANDLE=-12
	if errHandle != 0 {
		var mode uint32
		_, _, _ = getConsoleMode.Call(errHandle, uintptr(unsafe.Pointer(&mode)))
		mode |= ENABLE_VTP_OUTPUT | ENABLE_PROCESSED_OUTPUT | DISABLE_NEWLINE_AUTO_RET
		_, _, _ = setConsoleMode.Call(errHandle, uintptr(mode))
	}

	// Prefer not to override Windows Terminal font
	if os.Getenv("WT_SESSION") == "" {
		// Try Cascadia Mono first, then Consolas
		setConsoleFontWindows("Cascadia Mono", 16)
		setConsoleFontWindows("Consolas", 16)
	}
}

// setConsoleFontWindows attempts to switch to a Unicode-capable TrueType font.
func setConsoleFontWindows(face string, height int16) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getCurrentConsoleFontEx := kernel32.NewProc("GetCurrentConsoleFontEx")
	setCurrentConsoleFontEx := kernel32.NewProc("SetCurrentConsoleFontEx")

	type coord struct{ X, Y int16 }
	type consoleFontInfoEx struct {
		CbSize     uint32
		NFont      uint32
		DwFontSize coord
		FontFamily uint32
		FontWeight uint32
		FaceName   [32]uint16
	}

	h, _, _ := getStdHandle.Call(uintptr(^uint32(11) + 1)) // STD_OUTPUT_HANDLE
	if h == 0 {
		return
	}

	var info consoleFontInfoEx
	info.CbSize = uint32(unsafe.Sizeof(info))
	_, _, _ = getCurrentConsoleFontEx.Call(h, 0, uintptr(unsafe.Pointer(&info)))

	faceUTF16, err := syscall.UTF16FromString(face)
	if err != nil {
		return
	}
	for i := range info.FaceName {
		info.FaceName[i] = 0
	}
	copy(info.FaceName[:], faceUTF16)
	if height > 0 {
		info.DwFontSize.Y = height
	}
	if info.FontFamily == 0 {
		info.FontFamily = 0x36
	}
	if info.FontWeight == 0 {
		info.FontWeight = 400
	}
	_, _, _ = setCurrentConsoleFontEx.Call(h, 0, uintptr(unsafe.Pointer(&info)))
}
