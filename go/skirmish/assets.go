package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	xdraw "golang.org/x/image/draw"
)

//go:embed skirmish.png
var embeddedSkirmishPNG []byte

//go:embed skirmish.ico
var embeddedSkirmishICO []byte

// LoadHeaderImage loads skirmish.png either from embedded bytes or from disk.
func LoadHeaderImage() *ebiten.Image {
	// 1. Try embedded PNG first (guaranteed to be present in binary)
	if len(embeddedSkirmishPNG) > 0 {
		if img, err := png.Decode(bytes.NewReader(embeddedSkirmishPNG)); err == nil {
			return ebiten.NewImageFromImage(img)
		} else {
			log.Printf("failed to decode embedded skirmish.png: %v", err)
		}
	}

	// 2. Try disk files as fallback
	for _, filename := range []string{"skirmish.png", "icon.png"} {
		if data, err := os.ReadFile(filename); err == nil {
			if img, err := png.Decode(bytes.NewReader(data)); err == nil {
				return ebiten.NewImageFromImage(img)
			}
		}
	}

	// 3. Fallback procedural banner
	w, h := 640, 100
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			factor := float64(y) / float64(h)
			r := uint8(20 + factor*15)
			g := uint8(24 + factor*18)
			b := uint8(38 + factor*20)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	gold := color.RGBA{R: 212, G: 175, B: 55, A: 255}
	for x := 0; x < w; x++ {
		dst.SetRGBA(x, 0, gold)
		dst.SetRGBA(x, 1, gold)
		dst.SetRGBA(x, h-2, gold)
		dst.SetRGBA(x, h-1, gold)
	}

	return ebiten.NewImageFromImage(dst)
}

// LoadWindowIcons decodes skirmish.ico for crisp titlebar and taskbar icons.
func LoadWindowIcons() []image.Image {
	// 1. Try embedded skirmish.ico first
	if len(embeddedSkirmishICO) > 0 {
		if icons := decodeICO(embeddedSkirmishICO); len(icons) > 0 {
			return icons
		}
	}

	// 2. Try skirmish.ico from disk as fallback
	if data, err := os.ReadFile("skirmish.ico"); err == nil {
		if icons := decodeICO(data); len(icons) > 0 {
			return icons
		}
	}

	// 3. Fallback to PNG or procedural die icon if ICO missing
	var src image.Image
	if len(embeddedSkirmishPNG) > 0 {
		if img, err := png.Decode(bytes.NewReader(embeddedSkirmishPNG)); err == nil {
			src = img
		}
	}
	if src != nil {
		var icons []image.Image
		for _, size := range []int{16, 32, 48, 64, 128, 256} {
			icons = append(icons, scaleImage(src, size))
		}
		return icons
	}

	var icons []image.Image
	for _, size := range []int{16, 32, 48, 64} {
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		bg := color.RGBA{R: 195, G: 40, B: 40, A: 255}
		gold := color.RGBA{R: 240, G: 200, B: 60, A: 255}
		cx, cy := float64(size)/2, float64(size)/2
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := (float64(x) - cx) / (float64(size) * 0.45)
				dy := (float64(y) - cy) / (float64(size) * 0.45)
				if (dx*dx + dy*dy) <= 1.0 {
					img.SetRGBA(x, y, bg)
				}
			}
		}
		pipR := size / 6
		if pipR < 1 {
			pipR = 1
		}
		for py := -pipR; py <= pipR; py++ {
			for px := -pipR; px <= pipR; px++ {
				img.SetRGBA(int(cx)+px, int(cy)+py, gold)
			}
		}
		icons = append(icons, img)
	}
	return icons
}

// decodeICO parses Windows .ico files supporting both PNG frames and 32-bit BGRA DIB frames.
func decodeICO(data []byte) []image.Image {
	if len(data) < 6 {
		return nil
	}
	icoType := binary.LittleEndian.Uint16(data[2:4])
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if icoType != 1 || count <= 0 || len(data) < 6+count*16 {
		return nil
	}

	var result []image.Image
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	for i := 0; i < count; i++ {
		off := 6 + i*16
		w := int(data[off+0])
		if w == 0 {
			w = 256
		}
		h := int(data[off+1])
		if h == 0 {
			h = 256
		}
		size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		imgOff := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
		if imgOff < 0 || size <= 0 || imgOff+size > len(data) {
			continue
		}

		payload := data[imgOff : imgOff+size]

		// Case 1: Embedded PNG frame (e.g. 256x256 Vista+ icon)
		if len(payload) >= 8 && bytes.Equal(payload[:8], pngMagic) {
			if img, err := png.Decode(bytes.NewReader(payload)); err == nil {
				result = append(result, img)
			}
			continue
		}

		// Case 2: Standard 32-bit BGRA DIB frame (BITMAPINFOHEADER = 40 bytes)
		if len(payload) >= 40 {
			hdrSize := int(binary.LittleEndian.Uint32(payload[0:4]))
			bpp := int(binary.LittleEndian.Uint16(payload[14:16]))
			if hdrSize == 40 && bpp == 32 && len(payload) >= hdrSize+w*h*4 {
				nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
				pxData := payload[hdrSize:]
				for row := 0; row < h; row++ {
					// DIB rows are bottom-up
					destY := h - 1 - row
					for col := 0; col < w; col++ {
						idx := (row*w + col) * 4
						b := pxData[idx+0]
						g := pxData[idx+1]
						r := pxData[idx+2]
						a := pxData[idx+3]
						nrgba.SetNRGBA(col, destY, color.NRGBA{R: r, G: g, B: b, A: a})
					}
				}
				result = append(result, nrgba)
			}
		}
	}

	return result
}

func scaleImage(src image.Image, n int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
