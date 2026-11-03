//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"syscall"
)

func getImageFromWindowsClipboard() (image.Image, error) {
	script := `
Add-Type -AssemblyName System.Windows.Forms
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($img -ne $null) {
    $ms = New-Object System.IO.MemoryStream
    $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    [Console]::OpenStandardOutput().Write($ms.ToArray(), 0, $ms.Length)
} else {
    exit 1
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	if err != nil || outBuf.Len() == 0 {
		return nil, fmt.Errorf("no image found on clipboard")
	}

	img, err := png.Decode(&outBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode clipboard image: %v", err)
	}
	return img, nil
}
