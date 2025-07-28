//go:build windows

package main

import (
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows API functions loaded dynamically from kernel32.dll to ensure they are found.
var (
	kernel32                          = windows.NewLazySystemDLL("kernel32.dll")
	procGetNumberOfConsoleInputEvents = kernel32.NewProc("GetNumberOfConsoleInputEvents")
	procReadConsoleInput              = kernel32.NewProc("ReadConsoleInputW") // Use the Unicode (W) version
)

// We define these structs locally to avoid any "undefined" errors from the compiler.
const KEY_EVENT = 0x0001

type INPUT_RECORD struct {
	EventType uint16
	_         uint16 // Padding
	Event     [16]byte
}

type KEY_EVENT_RECORD struct {
	BKeyDown          int32
	WRepeatCount      uint16
	WVirtualKeyCode   uint16
	WVirtualScanCode  uint16
	UnicodeChar       uint16
	DwControlKeyState uint32
}

// cancellableRead attempts to read a single byte from stdin in a non-blocking,
// cancellable way that is safe for Windows. It polls the console input buffer
// for key-press events, allowing it to remain responsive to the 'done' channel.
func cancellableRead(done chan struct{}) (byte, error) {
	stdinHandle := windows.Handle(os.Stdin.Fd())
	for {
		select {
		case <-done:
			return 0, os.ErrClosed
		default:
			var events uint32
			r1, _, err := procGetNumberOfConsoleInputEvents.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&events)))
			if r1 == 0 {
				return 0, err // Call failed
			}

			if events > 0 {
				var records [1]INPUT_RECORD
				var n uint32
				r1, _, err = procReadConsoleInput.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&records[0])), 1, uintptr(unsafe.Pointer(&n)))
				if r1 == 0 { // Call failed
					// Propagate the error instead of looping, which could cause a busy-wait on a persistent error.
					return 0, err
				}

				record := records[0]
				if record.EventType == windows.KEY_EVENT {
					keyEvent := (*KEY_EVENT_RECORD)(unsafe.Pointer(&record.Event))
					if keyEvent.BKeyDown != 0 && keyEvent.UnicodeChar != 0 {
						return byte(keyEvent.UnicodeChar), nil
					}
				}
				continue
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
