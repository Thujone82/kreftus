//go:build !windows

package main

import "os/exec"

func hideSubprocessWindow(_ *exec.Cmd) {}

func initCLIConsole() bool { return false }
