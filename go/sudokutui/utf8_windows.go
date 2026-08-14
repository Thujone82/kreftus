//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleOutputCP      = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP            = kernel32.NewProc("SetConsoleCP")
	procGetStdHandle            = kernel32.NewProc("GetStdHandle")
	procGetConsoleMode          = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode          = kernel32.NewProc("SetConsoleMode")
	procGetCurrentConsoleFontEx = kernel32.NewProc("GetCurrentConsoleFontEx")
	procSetCurrentConsoleFontEx = kernel32.NewProc("SetCurrentConsoleFontEx")
)

const (
	utf8CodePage                    = 65001
	stdOutputHandle                 = ^uintptr(10) // STD_OUTPUT_HANDLE (-11)
	enableProcessedOutput           = 0x0001
	enableWrapAtEOLOutput           = 0x0002
	enableVirtualTerminalProcessing = 0x0004
)

type coord struct {
	X int16
	Y int16
}

// CONSOLE_FONT_INFOEX
type consoleFontInfoEx struct {
	cbSize     uint32
	nFont      uint32
	dwFontSize coord
	FontFamily uint32
	FontWeight uint32
	FaceName   [32]uint16
}

// Preferred monospace faces that include box-drawing and digit glyphs.
var consoleFontFaces = []string{
	"Cascadia Mono",
	"Cascadia Code",
	"Consolas",
	"Lucida Console",
}

// enableUTF8Console switches the Windows console to UTF-8 (CP65001),
// enables VT processing, and selects a Unicode-capable console font
// so box-drawing and gameplay glyphs render instead of replacement boxes.
func enableUTF8Console() {
	_, _, _ = procSetConsoleOutputCP.Call(uintptr(utf8CodePage))
	_, _, _ = procSetConsoleCP.Call(uintptr(utf8CodePage))

	h, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return
	}
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r != 0 {
		mode |= enableProcessedOutput | enableWrapAtEOLOutput | enableVirtualTerminalProcessing
		_, _, _ = procSetConsoleMode.Call(h, uintptr(mode))
	}
	setUnicodeConsoleFont(h)
}

func setUnicodeConsoleFont(h uintptr) {
	var cur consoleFontInfoEx
	cur.cbSize = uint32(unsafe.Sizeof(cur))
	_, _, _ = procGetCurrentConsoleFontEx.Call(h, 0, uintptr(unsafe.Pointer(&cur)))
	sizeY := cur.dwFontSize.Y
	if sizeY < 12 {
		sizeY = 16
	}

	for _, face := range consoleFontFaces {
		var info consoleFontInfoEx
		info.cbSize = uint32(unsafe.Sizeof(info))
		info.dwFontSize.Y = sizeY
		info.FontWeight = 400
		info.FontFamily = 54 // FF_MODERN | TMPF_TRUETYPE | TMPF_FIXED_PITCH-ish
		writeFaceName(&info, face)
		r, _, _ := procSetCurrentConsoleFontEx.Call(h, 0, uintptr(unsafe.Pointer(&info)))
		if r != 0 {
			return
		}
	}
}

func writeFaceName(info *consoleFontInfoEx, face string) {
	u, err := syscall.UTF16FromString(face)
	if err != nil {
		return
	}
	n := len(u)
	if n > len(info.FaceName) {
		n = len(info.FaceName)
	}
	copy(info.FaceName[:], u[:n])
}
