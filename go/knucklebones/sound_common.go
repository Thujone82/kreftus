package main

import (
	"os"
	"runtime"
	"strings"
)

type soundID int

const (
	sndNone soundID = iota
	sndClick
	sndPlace
	sndDestroy
	sndTurn
	sndWin
	sndLose
)

func (id soundID) tuiAudible() bool {
	switch id {
	case sndDestroy, sndWin, sndLose:
		return true
	}
	return false
}

// needPCMAudio reports whether synthesized PCM playback should be attempted for
// the active session. Local TUI sessions keep PCM enabled; only headless,
// remote SSH, or explicitly bypassed sessions fall back to terminal bells.
func needPCMAudio(goos string, ssh, sshCon bool, display, wayland, headlessEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(headlessEnv)) {
	case "1", "true", "yes":
		return false
	}
	if ssh || sshCon {
		return false
	}
	if goos == "linux" && strings.TrimSpace(display) == "" && strings.TrimSpace(wayland) == "" {
		return false
	}
	return true
}

func headlessAudio() bool {
	return !needPCMAudio(
		runtime.GOOS,
		remoteSSH(),
		sshLikeConsole(),
		os.Getenv("DISPLAY"),
		os.Getenv("WAYLAND_DISPLAY"),
		os.Getenv("KNUCKLEBONES_HEADLESS"),
	)
}
