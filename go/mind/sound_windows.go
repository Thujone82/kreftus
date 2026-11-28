//go:build windows

package main

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	winmm          = syscall.NewLazyDLL("winmm.dll")
	procPlaySoundW = winmm.NewProc("PlaySoundW")
)

const (
	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndMemory    = 0x0004
)

var (
	wavMu    sync.Mutex
	wavCache = map[soundID][]byte{}
)

func playPCM(id soundID) {
	wavMu.Lock()
	b, ok := wavCache[id]
	if !ok {
		if spec, found := soundSpecs[id]; found {
			b = synthWAV(spec.notes, spec.square)
			wavCache[id] = b
		}
	}
	wavMu.Unlock()
	if len(b) == 0 {
		return
	}
	_, _, _ = procPlaySoundW.Call(
		uintptr(unsafe.Pointer(&b[0])),
		0,
		sndMemory|sndAsync|sndNoDefault,
	)
}
