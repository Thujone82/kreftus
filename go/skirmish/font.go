package main

import (
	"bytes"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

type FontKind int

const (
	FontRegular FontKind = iota
	FontBold
	FontMono
)

type faceKey struct {
	kind  FontKind
	steps int // size rounded to 0.5px, stored as size*2
}

var (
	fontSources [3]*text.GoTextFaceSource
	faceCache   = map[faceKey]*text.GoTextFace{}
)

func initFonts() {
	faceCache = map[faceKey]*text.GoTextFace{}
	for i, ttf := range [][]byte{goregular.TTF, gobold.TTF, gomono.TTF} {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
		if err != nil {
			log.Fatalf("failed to parse font: %v", err)
		}
		fontSources[i] = src
	}
}

// GetFace returns a cached face. Sizes are rounded to half a pixel so dice
// numerals and labels reuse glyph images instead of rasterizing a new size
// every frame.
func GetFace(size float64, kind FontKind) *text.GoTextFace {
	if size < 1 {
		size = 1
	}
	if kind < 0 || int(kind) >= len(fontSources) {
		kind = FontRegular
	}
	steps := int(math.Round(size * 2))
	if steps < 2 {
		steps = 2
	}
	key := faceKey{kind: kind, steps: steps}
	if face, ok := faceCache[key]; ok {
		return face
	}
	face := &text.GoTextFace{
		Source: fontSources[kind],
		Size:   float64(steps) / 2,
	}
	faceCache[key] = face
	return face
}
