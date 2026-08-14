package main

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	colorWheelSize = 16
	flashDuration  = 600 * time.Millisecond
)

// Bright 16-step hue wheel — readable as title FG and as HUD/selection BG with black text.
var colorWheel = [colorWheelSize]tcell.Color{
	tcell.NewRGBColor(255, 77, 77),   // 0 red
	tcell.NewRGBColor(255, 122, 51),  // 1 orange-red
	tcell.NewRGBColor(255, 159, 26),  // 2 orange
	tcell.NewRGBColor(255, 209, 26),  // 3 gold
	tcell.NewRGBColor(245, 230, 77),  // 4 yellow
	tcell.NewRGBColor(184, 230, 77),  // 5 yellow-green
	tcell.NewRGBColor(92, 219, 92),   // 6 green
	tcell.NewRGBColor(46, 230, 160),  // 7 spring
	tcell.NewRGBColor(46, 217, 217),  // 8 cyan
	tcell.NewRGBColor(51, 184, 255),  // 9 sky
	tcell.NewRGBColor(107, 140, 255), // 10 periwinkle
	tcell.NewRGBColor(155, 123, 255), // 11 lavender
	tcell.NewRGBColor(196, 92, 255),  // 12 violet
	tcell.NewRGBColor(255, 92, 225),  // 13 magenta
	tcell.NewRGBColor(255, 92, 168),  // 14 pink
	tcell.NewRGBColor(255, 107, 122), // 15 rose
}

var (
	styleFlashOK  = tcell.StyleDefault.Foreground(tcell.ColorLime).Background(tcell.ColorBlack).Bold(true)
	styleFlashBad = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
)

func (g *game) wheelColor(delta int) tcell.Color {
	i := (g.accentIndex + delta) % colorWheelSize
	if i < 0 {
		i += colorWheelSize
	}
	return colorWheel[i]
}

func (g *game) accent() tcell.Color {
	return g.wheelColor(0)
}

func (g *game) accent2() tcell.Color {
	return g.wheelColor(5)
}

func (g *game) accent3() tcell.Color {
	return g.wheelColor(-5)
}

func (g *game) rotateAccent(dir int) {
	g.accentIndex = (g.accentIndex + dir) % colorWheelSize
	if g.accentIndex < 0 {
		g.accentIndex += colorWheelSize
	}
}

func (g *game) titleStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(g.accent()).Background(tcell.ColorBlack).Bold(true)
}

func (g *game) selectStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.accent()).Bold(true)
}

func (g *game) hudStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.accent()).Bold(true)
}

func (g *game) overlayStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.accent2())
}

func (g *game) overlayTitleStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(g.accent()).Background(g.accent2()).Bold(true)
}

func (g *game) dangerStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.accent3()).Bold(true)
}

func (g *game) cursorStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(g.cursorColor()).Background(tcell.ColorBlack).Bold(true)
}

func (g *game) cursorColor() tcell.Color {
	delta := 5 // same offset as accent2()
	if g.pencil {
		for n := 0; n < colorWheelSize; n++ {
			idx := (g.accentIndex + delta) % colorWheelSize
			if idx < 0 {
				idx += colorWheelSize
			}
			if !pencilCursorClash(idx) {
				break
			}
			delta++
		}
	}
	return g.wheelColor(delta)
}

func pencilCursorClash(i int) bool {
	// Gold and yellow sit on top of the light-yellow pencil grid.
	return i == 3 || i == 4
}

func (g *game) flashing() bool {
	return time.Now().Before(g.flashUntil)
}

func (g *game) borderStyle() tcell.Style {
	if g.flashing() {
		if g.flashOK {
			return styleFlashOK
		}
		return styleFlashBad
	}
	if g.pencil {
		return styleGridPencil
	}
	return styleGrid
}

func (g *game) startFlash(ok bool) {
	g.flashOK = ok
	g.flashUntil = time.Now().Add(flashDuration)
	go func() {
		time.Sleep(flashDuration)
		g.requestRedraw()
	}()
}

func (g *game) requestRedraw() {
	if g.redraw == nil {
		return
	}
	select {
	case g.redraw <- struct{}{}:
	default:
	}
}
