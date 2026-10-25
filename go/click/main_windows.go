//go:build windows

package main

import "os"

func main() {
	if err := runApp(); err != nil {
		messageBox(0, "Click", err.Error())
		os.Exit(1)
	}
}
