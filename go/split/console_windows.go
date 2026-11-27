//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procAttachConsole      = kernel32.NewProc("AttachConsole")
	procGetStdHandle       = kernel32.NewProc("GetStdHandle")
	procGetFileType        = kernel32.NewProc("GetFileType")
	procCreateFileW        = kernel32.NewProc("CreateFileW")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = kernel32.NewProc("SetConsoleCP")
	procGetConsoleMode     = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode     = kernel32.NewProc("SetConsoleMode")
)

const (
	attachParentProcess = ^uintptr(0)
	stdOutputHandle     = ^uintptr(10) // STD_OUTPUT_HANDLE (-11)
	stdErrorHandle      = ^uintptr(11) // STD_ERROR_HANDLE (-12)
	fileTypeUnknown     = 0
	cpUTF8              = 65001

	enableProcessedOutput   = 0x0001
	enableVTProcessing      = 0x0004
	disableNewlineAutoRet   = 0x0008
	genericReadWrite        = 0x80000000 | 0x40000000
	fileShareReadWrite      = 0x00000001 | 0x00000002
	openExisting            = 3
)

// hideSubprocessWindow configures Windows subprocess execution without flashing a console window.
func hideSubprocessWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}

// openSharedReadFile opens a file for reading with FILE_SHARE_DELETE enabled on Windows
// so Explorer or another process can rename or delete it without sharing violations.
func openSharedReadFile(path string) (*os.File, error) {
	p16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(
		p16,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

func stdHandleUsable(stdID uintptr) bool {
	h, _, _ := procGetStdHandle.Call(stdID)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return false
	}
	ft, _, _ := procGetFileType.Call(h)
	return ft != fileTypeUnknown
}

// initCLIConsole attaches a windowsgui binary to its parent console (if not already piped)
// and enables UTF-8 + ANSI Virtual Terminal processing.
func initCLIConsole() (attachedConsole bool) {
	if stdHandleUsable(stdOutputHandle) {
		enableUTF8AndVT(stdOutputHandle)
		enableUTF8AndVT(stdErrorHandle)
		return false
	}

	r, _, _ := procAttachConsole.Call(attachParentProcess)
	if r == 0 {
		return false
	}

	conOut, err := syscall.UTF16PtrFromString("CONOUT$")
	if err == nil {
		hOut, _, _ := procCreateFileW.Call(
			uintptr(unsafe.Pointer(conOut)),
			genericReadWrite,
			fileShareReadWrite,
			0,
			openExisting,
			0,
			0,
		)
		if hOut != 0 && hOut != uintptr(syscall.InvalidHandle) {
			os.Stdout = os.NewFile(hOut, "CONOUT$")
			os.Stderr = os.Stdout
			enableUTF8AndVTOnHandle(hOut)
		}
	}
	_, _, _ = procSetConsoleOutputCP.Call(uintptr(cpUTF8))
	_, _, _ = procSetConsoleCP.Call(uintptr(cpUTF8))
	return true
}

func enableUTF8AndVT(stdID uintptr) {
	_, _, _ = procSetConsoleOutputCP.Call(uintptr(cpUTF8))
	_, _, _ = procSetConsoleCP.Call(uintptr(cpUTF8))
	h, _, _ := procGetStdHandle.Call(stdID)
	if h != 0 && h != uintptr(syscall.InvalidHandle) {
		enableUTF8AndVTOnHandle(h)
	}
}

func enableUTF8AndVTOnHandle(h uintptr) {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r != 0 {
		mode |= enableProcessedOutput | enableVTProcessing | disableNewlineAutoRet
		_, _, _ = procSetConsoleMode.Call(h, uintptr(mode))
	}
}
