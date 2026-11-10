package main

import "github.com/gdamore/tcell/v2"

const (
	themeDark  = "dark"
	themeLight = "light"
)

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
}
