//go:build !(windows || darwin || (linux && cgo))

package main

import "errors"

const guiAvailable = false

func runGUI() error { return errors.New("the GUI is not available in this build") }
