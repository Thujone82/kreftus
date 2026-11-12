package main

import (
	"image/color"

	"github.com/gdamore/tcell/v2"
)

// rgba converts a palette color for the window front end.
func rgba(c tcell.Color) color.RGBA {
	r, g, b := c.RGB()
	if r < 0 {
		return color.RGBA{A: 255}
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

const (
	themeDark      = "dark"
	themeLight     = "light"
	themeContrast  = "contrast"
	themeParchment = "parchment"
)

var themeOrder = []string{themeDark, themeLight, themeContrast, themeParchment}

var themeNames = map[string]string{
	themeDark:      "Dark",
	themeLight:     "Light",
	themeContrast:  "High Contrast",
	themeParchment: "Parchment",
}

func validTheme(id string) bool {
	_, ok := themes[id]
	return ok
}

func themeLabel(id string) string {
	if s := themeNames[id]; s != "" {
		return s
	}
	return themeNames[themeDark]
}

func nextTheme(cur string) string {
	for i, id := range themeOrder {
		if id == cur {
			return themeOrder[(i+1)%len(themeOrder)]
		}
	}
	return themeOrder[0]
}

func paletteFor(id string) palette {
	if p, ok := themes[id]; ok {
		return p
	}
	return themes[themeDark]
}

type palette struct {
	bg, fg, dim, accent, title, frame tcell.Color
	win, lose                         tcell.Color

	dieBg, pip        tcell.Color // single die
	pairBg, pairPip   tcell.Color // two of a value in a column
	tripleBg, triPip  tcell.Color // three of a value
	emptyBg           tcell.Color // open slot
	ghostBg, ghostPip tcell.Color // preview of the die about to be placed
	doomBg, doomPip   tcell.Color // dice the preview would destroy
	lastPip           tcell.Color // pips of the most recent placement
}

var themes = map[string]palette{
	themeDark: {
		bg:       tcell.ColorBlack,
		fg:       tcell.ColorWhite,
		dim:      tcell.NewRGBColor(128, 128, 136),
		accent:   tcell.NewRGBColor(240, 184, 64),
		title:    tcell.NewRGBColor(226, 64, 64),
		frame:    tcell.NewRGBColor(96, 92, 104),
		win:      tcell.NewRGBColor(96, 208, 112),
		lose:     tcell.NewRGBColor(232, 88, 88),
		dieBg:    tcell.NewRGBColor(236, 228, 210),
		pip:      tcell.NewRGBColor(40, 30, 30),
		pairBg:   tcell.NewRGBColor(244, 196, 72),
		pairPip:  tcell.NewRGBColor(48, 32, 8),
		tripleBg: tcell.NewRGBColor(104, 164, 240),
		triPip:   tcell.NewRGBColor(16, 20, 48),
		emptyBg:  tcell.NewRGBColor(36, 36, 44),
		ghostBg:  tcell.NewRGBColor(92, 84, 70),
		ghostPip: tcell.NewRGBColor(244, 236, 214),
		doomBg:   tcell.NewRGBColor(200, 56, 56),
		doomPip:  tcell.ColorWhite,
		lastPip:  tcell.NewRGBColor(200, 36, 36),
	},
	themeLight: {
		bg:       tcell.NewRGBColor(246, 242, 234),
		fg:       tcell.NewRGBColor(32, 28, 28),
		dim:      tcell.NewRGBColor(120, 116, 112),
		accent:   tcell.NewRGBColor(176, 104, 0),
		title:    tcell.NewRGBColor(184, 32, 32),
		frame:    tcell.NewRGBColor(170, 164, 156),
		win:      tcell.NewRGBColor(24, 136, 48),
		lose:     tcell.NewRGBColor(192, 40, 40),
		dieBg:    tcell.NewRGBColor(58, 50, 46),
		pip:      tcell.NewRGBColor(246, 238, 220),
		pairBg:   tcell.NewRGBColor(214, 150, 20),
		pairPip:  tcell.NewRGBColor(40, 24, 0),
		tripleBg: tcell.NewRGBColor(44, 104, 200),
		triPip:   tcell.ColorWhite,
		emptyBg:  tcell.NewRGBColor(216, 208, 192),
		ghostBg:  tcell.NewRGBColor(150, 138, 120),
		ghostPip: tcell.ColorWhite,
		doomBg:   tcell.NewRGBColor(206, 48, 48),
		doomPip:  tcell.ColorWhite,
		lastPip:  tcell.NewRGBColor(255, 104, 96),
	},
	themeContrast: {
		bg:       tcell.ColorBlack,
		fg:       tcell.ColorWhite,
		dim:      tcell.NewRGBColor(204, 204, 214),
		accent:   tcell.NewRGBColor(255, 212, 0),
		title:    tcell.NewRGBColor(255, 84, 84),
		frame:    tcell.NewRGBColor(192, 192, 208),
		win:      tcell.NewRGBColor(56, 242, 120),
		lose:     tcell.NewRGBColor(255, 76, 76),
		dieBg:    tcell.ColorWhite,
		pip:      tcell.ColorBlack,
		pairBg:   tcell.NewRGBColor(255, 212, 0),
		pairPip:  tcell.ColorBlack,
		tripleBg: tcell.NewRGBColor(0, 224, 255),
		triPip:   tcell.ColorBlack,
		emptyBg:  tcell.NewRGBColor(28, 28, 36),
		ghostBg:  tcell.NewRGBColor(92, 92, 114),
		ghostPip: tcell.ColorWhite,
		doomBg:   tcell.NewRGBColor(240, 32, 32),
		doomPip:  tcell.ColorWhite,
		lastPip:  tcell.NewRGBColor(220, 0, 110),
	},
	themeParchment: {
		bg:       tcell.NewRGBColor(239, 227, 200),
		fg:       tcell.NewRGBColor(43, 29, 20),
		dim:      tcell.NewRGBColor(128, 108, 86),
		accent:   tcell.NewRGBColor(178, 86, 18),
		title:    tcell.NewRGBColor(158, 31, 31),
		frame:    tcell.NewRGBColor(176, 156, 126),
		win:      tcell.NewRGBColor(40, 124, 58),
		lose:     tcell.NewRGBColor(172, 34, 34),
		dieBg:    tcell.NewRGBColor(62, 43, 32),
		pip:      tcell.NewRGBColor(250, 242, 222),
		pairBg:   tcell.NewRGBColor(216, 142, 28),
		pairPip:  tcell.NewRGBColor(38, 22, 8),
		tripleBg: tcell.NewRGBColor(46, 95, 168),
		triPip:   tcell.NewRGBColor(250, 242, 222),
		emptyBg:  tcell.NewRGBColor(216, 200, 172),
		ghostBg:  tcell.NewRGBColor(158, 138, 110),
		ghostPip: tcell.NewRGBColor(250, 242, 222),
		doomBg:   tcell.NewRGBColor(186, 38, 38),
		doomPip:  tcell.ColorWhite,
		lastPip:  tcell.NewRGBColor(238, 86, 68),
	},
}
