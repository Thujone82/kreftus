//go:build !windows

package main

import (
	"os"
	"os/exec"
)

func hideSubprocessWindow(_ *exec.Cmd) {}

func openSharedReadFile(path string) (*os.File, error) {
	return os.Open(path)
}

func initCLIConsole() bool { return false }
