package main

type rgb struct {
	R, G, B uint8
}

type boxStyle struct {
	TL, TR, BL, BR, H, V rune
	LT, RT, TT, BT, X    rune
	IH, IV, ILT, IRT     rune
	ITT, IBT             rune
}

type theme struct {
	ID, Name                               string
	BG, FG, Dim, Frame, Accent, Title      rgb
	Win, Lose                              rgb
	Pegs                                   [numColors]rgb
	Exact, Partial                         rgb
	ExactGlyph, PartialGlyph, EmptyFbGlyph rune
	EmptySlot                              rune
	Cover                                  rune
	Tiles                                  bool
	Box                                    boxStyle
}

var (
	boxDouble = boxStyle{
		TL: '╔', TR: '╗', BL: '╚', BR: '╝', H: '═', V: '║',
		LT: '╠', RT: '╣', TT: '╦', BT: '╩', X: '╬',
		IH: '─', IV: '│', ILT: '╟', IRT: '╢', ITT: '╤', IBT: '╧',
	}
	boxHeavy = boxStyle{
		TL: '┏', TR: '┓', BL: '┗', BR: '┛', H: '━', V: '┃',
		LT: '┣', RT: '┫', TT: '┳', BT: '┻', X: '╋',
		IH: '─', IV: '│', ILT: '┠', IRT: '┨', ITT: '┯', IBT: '┷',
	}
	boxRounded = boxStyle{
		TL: '╭', TR: '╮', BL: '╰', BR: '╯', H: '─', V: '│',
		LT: '├', RT: '┤', TT: '┬', BT: '┴', X: '┼',
		IH: '─', IV: '│', ILT: '├', IRT: '┤', ITT: '┬', IBT: '┴',
	}
)

const defaultThemeID = "dark"

var themeOrder = []string{"dark", "light", "neon", "ocean", "wood", "contrast"}

var themes = map[string]theme{
	"dark": {
		ID:           "dark",
		Name:         "Dark",
		BG:           rgb{12, 12, 16},
		FG:           rgb{220, 220, 225},
		Dim:          rgb{110, 110, 120},
		Frame:        rgb{90, 90, 110},
		Accent:       rgb{240, 190, 70},
		Title:        rgb{235, 70, 70},
		Win:          rgb{46, 204, 113},
		Lose:         rgb{235, 70, 70},
		Pegs:         [numColors]rgb{{230, 57, 70}, {46, 204, 113}, {52, 120, 246}, {0, 200, 220}, {200, 80, 220}, {245, 205, 50}},
		Exact:        rgb{46, 204, 113},
		Partial:      rgb{240, 170, 40},
		ExactGlyph:   '●',
		PartialGlyph: '●',
		EmptyFbGlyph: '·',
		EmptySlot:    '○',
		Cover:        '▒',
		Tiles:        false,
		Box:          boxDouble,
	},
	"light": {
		ID:           "light",
		Name:         "Light",
		BG:           rgb{245, 243, 238},
		FG:           rgb{30, 30, 35},
		Dim:          rgb{140, 138, 135},
		Frame:        rgb{170, 165, 160},
		Accent:       rgb{180, 100, 0},
		Title:        rgb{190, 30, 30},
		Win:          rgb{20, 150, 70},
		Lose:         rgb{190, 30, 30},
		Pegs:         [numColors]rgb{{200, 30, 45}, {20, 150, 70}, {30, 90, 210}, {0, 140, 170}, {160, 40, 170}, {210, 160, 0}},
		Exact:        rgb{20, 150, 70},
		Partial:      rgb{210, 140, 0},
		ExactGlyph:   '●',
		PartialGlyph: '●',
		EmptyFbGlyph: '·',
		EmptySlot:    '○',
		Cover:        '▒',
		Tiles:        false,
		Box:          boxRounded,
	},
	"neon": {
		ID:           "neon",
		Name:         "Neon",
		BG:           rgb{22, 10, 38},
		FG:           rgb{240, 220, 255},
		Dim:          rgb{130, 100, 160},
		Frame:        rgb{255, 60, 200},
		Accent:       rgb{0, 240, 255},
		Title:        rgb{255, 60, 200},
		Win:          rgb{60, 255, 140},
		Lose:         rgb{255, 50, 90},
		Pegs:         [numColors]rgb{{255, 50, 90}, {60, 255, 140}, {70, 120, 255}, {0, 240, 255}, {255, 60, 220}, {255, 230, 60}},
		Exact:        rgb{60, 255, 140},
		Partial:      rgb{255, 230, 60},
		ExactGlyph:   '●',
		PartialGlyph: '●',
		EmptyFbGlyph: '·',
		EmptySlot:    '○',
		Cover:        '▒',
		Tiles:        false,
		Box:          boxHeavy,
	},
	"ocean": {
		ID:           "ocean",
		Name:         "Ocean",
		BG:           rgb{8, 28, 48},
		FG:           rgb{210, 230, 240},
		Dim:          rgb{90, 130, 160},
		Frame:        rgb{40, 110, 160},
		Accent:       rgb{120, 220, 230},
		Title:        rgb{120, 220, 230},
		Win:          rgb{100, 220, 140},
		Lose:         rgb{255, 99, 99},
		Pegs:         [numColors]rgb{{255, 99, 99}, {100, 220, 140}, {90, 150, 255}, {80, 230, 230}, {220, 120, 230}, {255, 215, 100}},
		Exact:        rgb{100, 220, 140},
		Partial:      rgb{255, 215, 100},
		ExactGlyph:   '●',
		PartialGlyph: '●',
		EmptyFbGlyph: '·',
		EmptySlot:    '◦',
		Cover:        '▒',
		Tiles:        false,
		Box:          boxRounded,
	},
	"wood": {
		ID:           "wood",
		Name:         "Wood Board",
		BG:           rgb{58, 36, 22},
		FG:           rgb{240, 225, 200},
		Dim:          rgb{160, 130, 100},
		Frame:        rgb{190, 140, 80},
		Accent:       rgb{230, 180, 90},
		Title:        rgb{240, 200, 120},
		Win:          rgb{120, 200, 110},
		Lose:         rgb{230, 90, 70},
		Pegs:         [numColors]rgb{{220, 50, 50}, {60, 170, 80}, {50, 100, 200}, {60, 180, 190}, {170, 70, 170}, {240, 200, 60}},
		Exact:        rgb{250, 245, 235},
		Partial:      rgb{250, 245, 235},
		ExactGlyph:   '●',
		PartialGlyph: '○',
		EmptyFbGlyph: '·',
		EmptySlot:    '○',
		Cover:        '▓',
		Tiles:        false,
		Box:          boxDouble,
	},
	"contrast": {
		ID:           "contrast",
		Name:         "High Contrast",
		BG:           rgb{0, 0, 0},
		FG:           rgb{255, 255, 255},
		Dim:          rgb{200, 200, 200},
		Frame:        rgb{255, 255, 255},
		Accent:       rgb{255, 255, 0},
		Title:        rgb{255, 255, 0},
		Win:          rgb{0, 255, 0},
		Lose:         rgb{255, 60, 60},
		Pegs:         [numColors]rgb{{255, 0, 0}, {0, 230, 0}, {60, 110, 255}, {0, 255, 255}, {255, 0, 255}, {255, 255, 0}},
		Exact:        rgb{255, 255, 255},
		Partial:      rgb{255, 255, 255},
		ExactGlyph:   '●',
		PartialGlyph: '○',
		EmptyFbGlyph: '·',
		EmptySlot:    '○',
		Cover:        '?',
		Tiles:        true,
		Box:          boxHeavy,
	},
}

func validTheme(id string) bool {
	_, ok := themes[id]
	return ok
}

func themeByID(id string) theme {
	if t, ok := themes[id]; ok {
		return t
	}
	return themes[defaultThemeID]
}

func nextTheme(cur string) string {
	for i, id := range themeOrder {
		if id == cur {
			return themeOrder[(i+1)%len(themeOrder)]
		}
	}
	return themeOrder[0]
}
