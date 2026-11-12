//go:build !(windows || darwin || (linux && cgo))

package main

func playPCMSound(soundID) bool { return false }
