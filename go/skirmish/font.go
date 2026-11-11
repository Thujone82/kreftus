package main

import (
	"bytes"
	"log"

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

var fontSources [3]*text.GoTextFaceSource

func initFonts() {
	for i, ttf := range [][]byte{goregular.TTF, gobold.TTF, gomono.TTF} {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
		if err != nil {
			log.Fatalf("failed to parse font: %v", err)
		}
		fontSources[i] = src
	}
}

func GetFace(size float64, kind FontKind) *text.GoTextFace {
	return &text.GoTextFace{
		Source: fontSources[kind],
		Size:   size,
	}
}
