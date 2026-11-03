//go:build !windows

package main

import (
	"fmt"
	"image"
)

func getImageFromWindowsClipboard() (image.Image, error) {
	return nil, fmt.Errorf("clipboard image paste is only available on Windows")
}
