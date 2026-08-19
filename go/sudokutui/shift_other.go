//go:build !windows

package main

func shiftPollable() bool { return false }

func shiftHeld() bool { return false }

func startShiftWatch() {}

func stopShiftWatch() {}

func keypadOrigin() bool { return false }
