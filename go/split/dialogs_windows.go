//go:build windows

package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var (
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
)

const (
	ofnOverwritePrompt = 0x00000002
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnNoChangeDir     = 0x00000008
	ofnExplorer        = 0x00080000
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        unsafe.Pointer
	dwReserved        uint32
	flagsEx           uint32
}

func utf16Filter(pairs ...string) []uint16 {
	var raw strings.Builder
	for _, p := range pairs {
		raw.WriteString(p)
		raw.WriteByte(0)
	}
	raw.WriteByte(0)
	s := raw.String()
	u16 := make([]uint16, len(s))
	for i := 0; i < len(s); i++ {
		u16[i] = uint16(s[i])
	}
	return u16
}

// OpenVideoDialog displays the native Windows file picker for selecting a video file.
func OpenVideoDialog(owner uintptr, initialDir string) (string, bool) {
	fileBuf := make([]uint16, 4096)
	filter := utf16Filter(
		"Video Files (*.mp4;*.mkv;*.mov;*.webm;*.avi;*.ts;*.m4v;*.flv;*.wmv)",
		"*.mp4;*.mkv;*.mov;*.webm;*.avi;*.ts;*.m4v;*.flv;*.wmv",
		"All Files (*.*)",
		"*.*",
	)
	title := utf16Ptr("Open Video File")
	var dirPtr *uint16
	if initialDir != "" {
		dirPtr = utf16Ptr(initialDir)
	}

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		hwndOwner:       owner,
		lpstrFilter:     &filter[0],
		nFilterIndex:    1,
		lpstrFile:       &fileBuf[0],
		nMaxFile:        uint32(len(fileBuf)),
		lpstrInitialDir: dirPtr,
		lpstrTitle:      title,
		flags:           ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir,
	}

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	runtime.KeepAlive(dirPtr)
	runtime.KeepAlive(&fileBuf[0])
	if ret == 0 {
		return "", false
	}
	return syscall.UTF16ToString(fileBuf), true
}

// SaveVideoDialog displays the native Windows Save As dialog pre-filled with directory & suggested name.
func SaveVideoDialog(owner uintptr, titleText, initialDir, defaultName, ext string, promptOverwrite bool) (string, bool) {
	fileBuf := make([]uint16, 4096)
	if defaultName != "" {
		if u16, err := syscall.UTF16FromString(defaultName); err == nil {
			n := len(u16)
			if n > len(fileBuf)-1 {
				n = len(fileBuf) - 1
			}
			copy(fileBuf[:n], u16[:n])
		}
	}

	cleanExt := strings.TrimPrefix(strings.ToLower(ext), ".")
	extFilterLabel := "Video Files (*." + cleanExt + ")"
	extFilterPat := "*." + cleanExt
	if cleanExt == "" {
		extFilterLabel = "All Files (*.*)"
		extFilterPat = "*.*"
	}
	filter := utf16Filter(
		extFilterLabel, extFilterPat,
		"All Files (*.*)", "*.*",
	)
	title := utf16Ptr(titleText)
	var dirPtr *uint16
	if initialDir != "" {
		dirPtr = utf16Ptr(initialDir)
	}
	var defExtPtr *uint16
	if cleanExt != "" {
		defExtPtr = utf16Ptr(cleanExt)
	}

	flags := uint32(ofnExplorer | ofnPathMustExist | ofnNoChangeDir)
	if promptOverwrite {
		flags |= ofnOverwritePrompt
	}

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		hwndOwner:       owner,
		lpstrFilter:     &filter[0],
		nFilterIndex:    1,
		lpstrFile:       &fileBuf[0],
		nMaxFile:        uint32(len(fileBuf)),
		lpstrInitialDir: dirPtr,
		lpstrTitle:      title,
		lpstrDefExt:     defExtPtr,
		flags:           flags,
	}

	ret, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	runtime.KeepAlive(dirPtr)
	runtime.KeepAlive(defExtPtr)
	runtime.KeepAlive(&fileBuf[0])
	if ret == 0 {
		return "", false
	}
	chosen := syscall.UTF16ToString(fileBuf)
	if filepath.Ext(chosen) == "" && cleanExt != "" {
		chosen += "." + cleanExt
	}
	return chosen, true
}
