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
	procGetConsoleWindow        = kernel32.NewProc("GetConsoleWindow")
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

type consoleFontInfoEx struct {
	cbSize     uint32
	nFont      uint32
	dwFontSize coord
	FontFamily uint32
	FontWeight uint32
	FaceName   [32]uint16
}

var consoleFontFaces = []string{
	"Cascadia Mono",
	"Cascadia Code",
	"Consolas",
	"Lucida Console",
}

func consoleWindow() uintptr {
	h, _, _ := procGetConsoleWindow.Call()
	return h
}

// enableUTF8Console switches the Windows console to UTF-8 (CP65001),
// enables VT processing, and selects a Unicode-capable console font
// so box-drawing and gameplay glyphs render instead of replacement boxes.
// Font changes are skipped over SSH, where there is no interactive console UI.
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
	if consoleWindow() != 0 && !remoteSSH() {
		setUnicodeConsoleFont(h)
	}
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
		info.FontFamily = 54
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
