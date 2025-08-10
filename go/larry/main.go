package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/gdamore/tcell/v2"
)

type lane struct {
	y           int
	speedTicks  int
	dirRight    bool
	cars        []int // leftmost x for each vehicle in this lane
	width       int
	tickCounter int
	length      int         // vehicle length in cells
	glyph       []rune      // glyphs to render per cell (same length as length)
	color       tcell.Color // per-lane vehicle color
}

type theme struct {
	bg         tcell.Color
	fg         tcell.Color
	road       tcell.Color
	river      tcell.Color
	safe       tcell.Color
	frog       tcell.Color
	carSmall   tcell.Color
	carRegular tcell.Color
	carSemi    tcell.Color
	log        tcell.Color
	goal       tcell.Color
}

type game struct {
	screen tcell.Screen
	width  int
	height int

	level       int
	score       int
	topScore    int
	lives       int
	frogX       int
	frogY       int
	highestY    int
	hudY        int
	lanes       []lane
	safeTopY    int
	safeBottomY int
	safeRow     []bool
	rng         *rand.Rand
	theme       theme
	paused      bool
}

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := s.Init(); err != nil {
		panic(err)
	}
	defer s.Fini()
	s.Clear()
	s.HideCursor()

	g := &game{screen: s, rng: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))}
	g.initLevel(1)

	events := make(chan tcell.Event, 32)
	go func() {
		for {
			events <- s.PollEvent()
		}
	}()

	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()

	for {
		select {
		case ev := <-events:
			switch e := ev.(type) {
			case *tcell.EventResize:
				g.resize()
			case *tcell.EventKey:
				if handleQuit(e) {
					return
				}
				g.handleInput(e)
			}
		case <-tick.C:
			g.update()
			g.render()
		}
	}
}

func (g *game) resize() {
	// Recreate the world on resize to keep HUD/top/bottom shoulders correct
	g.width, g.height = g.screen.Size()
	if g.width <= 0 || g.height <= 0 {
		return
	}
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	// Respawn Larry to bottom safe shoulder and re-center horizontally
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.createLanes()
}

func (g *game) initLevel(level int) {
	g.level = level
	g.width, g.height = g.screen.Size()
	// Lives/score are set on first game start; keep values across levels.
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.theme = themeForLevel(level)
	g.createLanes()
}

func (g *game) nextLevel() {
	g.level++
	if g.level > 9 {
		g.level = 1
	}
	// Keep score/lives, reposition frog
	g.width, g.height = g.screen.Size()
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	// Reward: extra life each cleared level
	g.lives++
	g.theme = themeForLevel(g.level)
	g.createLanes()
}

func (g *game) createLanes() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	g.lanes = g.lanes[:0]
	g.safeRow = make([]bool, h)
	// shoulders are always safe
	if h > 0 {
		g.safeRow[0] = true
	}
	if h > 1 {
		g.safeRow[h-1] = true
	}
	// Generate roads: 4-6 lanes in one direction, then a safe gap of 1-3 rows, then flip direction.
	// Playfield between safeTopY and safeBottomY; HUD is at row 0.
	y := g.safeTopY + 1
	dirRight := g.rng.IntN(2) == 0
	for y < h-1 {
		// Road segment
		lanesThisRoad := 4 + g.rng.IntN(3) // 4..6
		if lanesThisRoad > 8 {
			lanesThisRoad = 8
		}
		// Adjust density and speed by level
		densityFactor := 0.5 + 0.05*float64(max(0, g.level-1)) // 0.5 at L1, +5% each level
		speedFactor := 0.67 + 0.05*float64(max(0, g.level-1))  // ~33% slower at L1, +5% each level
		if densityFactor > 1.5 {
			densityFactor = 1.5
		}
		if speedFactor > 1.25 {
			speedFactor = 1.25
		}

		for li := 0; li < lanesThisRoad && y < h-1; li++ {
			// Vehicle class selection per lane
			vehType := g.rng.IntN(3) // 0 compact, 1 regular, 2 semi
			minSpd, maxSpd := 3, 5
			color := g.theme.carSmall
			var glyph []rune
			if vehType == 0 {
				minSpd, maxSpd = 3, 5
				color = g.theme.carSmall
				if dirRight {
					glyph = []rune{'=', '>'} // carSmall '=>'
				} else {
					glyph = []rune{'<', '='} // carSmall '<='
				}
			} else if vehType == 1 {
				minSpd, maxSpd = 2, 4
				color = g.theme.carRegular
				if dirRight {
					glyph = []rune{'<', '#', '>'} // carRegular '<#>' (same both ways visually)
				} else {
					glyph = []rune{'<', '#', '>'}
				}
			} else { // vehType == 2 semi
				minSpd, maxSpd = 1, 3
				color = g.theme.carSemi
				if dirRight {
					glyph = []rune{'#', '#', '#', '#', '>'} // carSemi '####>'
				} else {
					glyph = []rune{'<', '#', '#', '#', '#'} // carSemi '<####'
				}
			}
			length := len(glyph)
			desired := minSpd + g.rng.IntN(maxSpd-minSpd+1)
			baseTicks := max(1, 7-desired) // map 1..5 to slower..faster tick counts
			speed := int(math.Round(float64(baseTicks) / speedFactor))
			if speed < 1 {
				speed = 1
			}

			// Base gap scales with densityFactor (more density -> smaller gaps)
			baseGap := int(math.Round(float64(max(2*length, 6)) / densityFactor))
			if baseGap < length+1 {
				baseGap = length + 1
			}
			num := max(1, int(float64(w)/(float64(length+baseGap))))
			positions := make([]int, 0, num)
			pos := g.rng.IntN(max(1, w))
			for k := 0; k < num; k++ {
				positions = append(positions, pos%max(1, w))
				pos += length + baseGap + g.rng.IntN(4)
			}
			g.lanes = append(g.lanes, lane{y: y, speedTicks: speed, dirRight: dirRight, cars: positions, width: w, length: length, glyph: glyph, color: color})
			if y >= 0 && y < h {
				g.safeRow[y] = false
			}
			y++
		}
		// Safe gap 1-3 lines
		gap := 1 + g.rng.IntN(3)
		for gi := 0; gi < gap && y < g.safeBottomY; gi++ {
			if y >= 0 && y < h {
				g.safeRow[y] = true
			}
			y++
		}
		// Flip road direction
		dirRight = !dirRight
	}
}

func (g *game) handleInput(e *tcell.EventKey) {
	// Toggle pause on Space
	if e.Key() == tcell.KeyRune && e.Rune() == ' ' {
		g.paused = !g.paused
		return
	}
	if g.paused {
		return
	}
	switch e.Key() {
	case tcell.KeyLeft:
		g.frogX--
	case tcell.KeyRight:
		g.frogX++
	case tcell.KeyUp:
		g.frogY--
		if g.frogY < g.highestY {
			g.score += (g.highestY - g.frogY) * 10 // per-line bonus when advancing upward
			g.highestY = g.frogY
			if g.score > g.topScore {
				g.topScore = g.score
			}
		}
	case tcell.KeyDown:
		g.frogY++
	default:
		switch e.Rune() {
		case 'a', 'A':
			g.frogX--
		case 'd', 'D':
			g.frogX++
		case 'w', 'W':
			g.frogY--
			if g.frogY < g.highestY {
				g.score += (g.highestY - g.frogY) * 10
				g.highestY = g.frogY
				if g.score > g.topScore {
					g.topScore = g.score
				}
			}
		case 's', 'S':
			g.frogY++
		}
	}
	g.clampFrog()
}

func (g *game) clampFrog() {
	if g.frogX < 0 {
		g.frogX = 0
	}
	if g.frogX >= g.width {
		g.frogX = max(0, g.width-1)
	}
	if g.frogY < 0 {
		g.frogY = 0
	}
	if g.frogY >= g.height {
		g.frogY = max(0, g.height-1)
	}
}

func (g *game) update() {
	if g.paused {
		return
	}
	// Advance lanes
	for i := range g.lanes {
		ln := &g.lanes[i]
		ln.tickCounter++
		if ln.tickCounter >= ln.speedTicks {
			ln.tickCounter = 0
			for j := range ln.cars {
				if ln.dirRight {
					ln.cars[j] = (ln.cars[j] + 1) % max(1, ln.width)
				} else {
					ln.cars[j] = (ln.cars[j] - 1 + max(1, ln.width)) % max(1, ln.width)
				}
			}
		}
	}

	// Collision detection with lanes (ignore safe rows)
	isSafe := g.frogY >= 0 && g.frogY < len(g.safeRow) && g.safeRow[g.frogY]
	if !isSafe {
		for _, ln := range g.lanes {
			if ln.y == g.frogY {
				for _, cx := range ln.cars {
					if g.frogX >= cx && g.frogX < cx+ln.length {
						// Hit! Lose a life
						g.lives--
						if g.lives <= 0 {
							g.gameOverFlash()
							// Reset whole game
							g.lives = 3
							g.score = 0
							g.level = 1
							g.theme = themeForLevel(g.level)
							g.createLanes()
							g.frogX = g.width / 2
							g.frogY = g.safeBottomY
							g.highestY = g.frogY
						} else {
							// Show a brief death message and respawn at bottom safe row
							g.youDiedFlash()
							g.frogX = g.width / 2
							g.frogY = g.safeBottomY
							g.highestY = g.frogY
						}
						break
					}
				}
				break
			}
		}
	}

	// Reached goal at top safe row
	if g.frogY == g.safeTopY {
		g.score += 100 * g.level
		if g.score > g.topScore {
			g.topScore = g.score
		}
		g.nextLevel()
	}
}

func (g *game) render() {
	s := g.screen
	s.Clear()
	w, h := g.width, g.height

	// Background fill (safe rows visually distinct)
	for y := 0; y < h; y++ {
		var bg tcell.Color
		if y == g.safeTopY {
			bg = g.theme.goal
		} else if y == g.safeBottomY || (y >= 0 && y < len(g.safeRow) && g.safeRow[y]) {
			bg = g.theme.safe
		} else if y%2 == 0 {
			bg = g.theme.road
		} else {
			bg = g.theme.river
		}
		st := tcell.StyleDefault.Background(bg)
		for x := 0; x < w; x++ {
			s.SetContent(x, y, ' ', nil, st)
		}
	}

	// Draw lanes' vehicles with length and glyphs
	for _, ln := range g.lanes {
		st := tcell.StyleDefault.Foreground(ln.color)
		for _, left := range ln.cars {
			for dx := 0; dx < ln.length; dx++ {
				x := left + dx
				if x >= 0 && x < w && ln.y >= 0 && ln.y < h {
					ch := '>'
					if dx < len(ln.glyph) {
						ch = ln.glyph[dx]
					}
					s.SetContent(x, ln.y, ch, nil, st)
				}
			}
		}
	}

	// Draw HUD (top status bar on row 0) with right-aligned TopScore
	left := fmt.Sprintf("Score:%d  Level:%d  Lives:%d  (Space:Pause, Q/Esc)", g.score, g.level, g.lives)
	right := fmt.Sprintf("Top:%d", g.topScore)
	hudLine := left
	if len(left)+1+len(right) < w {
		pad := w - len(left) - len(right)
		if pad < 1 {
			pad = 1
		}
		hudLine = left + spaces(pad) + right
	}
	drawText(s, 0, 0, hudLine, tcell.StyleDefault.Foreground(g.theme.fg).Background(g.theme.safe))

	// Draw Larry as a green '@' for wide-compat terminals
	frogStyle := tcell.StyleDefault.Foreground(g.theme.frog).Bold(true)
	s.SetContent(g.frogX, g.frogY, '@', nil, frogStyle)

	// Ensure pause overlay is drawn last, on top of vehicles and frog
	if g.paused {
		g.drawPauseOverlay()
	}

	s.Show()
}

func (g *game) gameOverFlash() {
	st := tcell.StyleDefault.Background(tcell.ColorMaroon)
	for i := 0; i < 3; i++ {
		for y := 0; y < g.height; y++ {
			for x := 0; x < g.width; x++ {
				g.screen.SetContent(x, y, ' ', nil, st)
			}
		}
		drawCentered(g.screen, g.width/2, g.height/2, "Game Over!", tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorMaroon).Bold(true))
		g.screen.Show()
		time.Sleep(200 * time.Millisecond)
	}
}

func (g *game) youDiedFlash() {
	st := tcell.StyleDefault.Background(tcell.ColorDarkRed)
	for i := 0; i < 2; i++ {
		for y := 0; y < g.height; y++ {
			for x := 0; x < g.width; x++ {
				g.screen.SetContent(x, y, ' ', nil, st)
			}
		}
		drawCentered(g.screen, g.width/2, g.height/2, "You Died!", tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkRed).Bold(true))
		g.screen.Show()
		time.Sleep(150 * time.Millisecond)
	}
}

func handleQuit(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlC {
		return true
	}
	r := e.Rune()
	return r == 'q' || r == 'Q'
}

func drawText(s tcell.Screen, x, y int, text string, st tcell.Style) {
	for i, ch := range text {
		s.SetContent(x+i, y, ch, nil, st)
	}
}

func drawCentered(s tcell.Screen, cx, cy int, text string, st tcell.Style) {
	x := cx - len([]rune(text))/2
	drawText(s, x, cy, text, st)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

func (g *game) drawPauseOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	title := "PAUSED"
	y0 := h/2 - 1
	if y0 < 0 {
		y0 = 0
	}
	if y0+2 >= h {
		y0 = max(0, h-3)
	}
	// Use Larry's color for the banner background for strong contrast
	st := tcell.StyleDefault.Background(g.theme.frog).Foreground(tcell.ColorBlack).Bold(true)
	for dy := 0; dy < 3; dy++ {
		drawText(g.screen, 0, y0+dy, spaces(w), st)
	}
	drawCentered(g.screen, w/2, y0+1, title, st)
}

func themeForLevel(level int) theme {
	palettes := []theme{
		{bg: tcell.ColorReset, fg: tcell.ColorWhite, road: tcell.ColorGray, river: tcell.ColorNavy, safe: tcell.ColorDarkOliveGreen, frog: tcell.ColorGreen, carSmall: tcell.ColorLightSalmon, carRegular: tcell.ColorOrangeRed, carSemi: tcell.ColorTomato, log: tcell.ColorSandyBrown, goal: tcell.ColorDarkCyan},
		{bg: tcell.ColorBlack, fg: tcell.ColorLightCyan, road: tcell.ColorDarkSlateGray, river: tcell.ColorBlue, safe: tcell.ColorDarkGreen, frog: tcell.ColorLawnGreen, carSmall: tcell.ColorLightSkyBlue, carRegular: tcell.ColorSteelBlue, carSemi: tcell.ColorRoyalBlue, log: tcell.ColorBurlyWood, goal: tcell.ColorDarkTurquoise},
		{bg: tcell.ColorBlack, fg: tcell.ColorWhite, road: tcell.ColorDimGray, river: tcell.ColorDarkBlue, safe: tcell.ColorDarkOliveGreen, frog: tcell.ColorChartreuse, carSmall: tcell.ColorPlum, carRegular: tcell.ColorMediumVioletRed, carSemi: tcell.ColorDeepPink, log: tcell.ColorPeru, goal: tcell.ColorTeal},
		{bg: tcell.ColorBlack, fg: tcell.ColorSilver, road: tcell.ColorGray, river: tcell.ColorDarkSlateBlue, safe: tcell.ColorDarkGreen, frog: tcell.ColorGreenYellow, carSmall: tcell.ColorKhaki, carRegular: tcell.ColorGoldenrod, carSemi: tcell.ColorSaddleBrown, log: tcell.ColorTan, goal: tcell.ColorCadetBlue},
		{bg: tcell.ColorBlack, fg: tcell.ColorWhite, road: tcell.ColorGray, river: tcell.ColorRoyalBlue, safe: tcell.ColorDarkOliveGreen, frog: tcell.ColorSpringGreen, carSmall: tcell.ColorLightGreen, carRegular: tcell.ColorSeaGreen, carSemi: tcell.ColorDarkGreen, log: tcell.ColorSandyBrown, goal: tcell.ColorSteelBlue},
	}
	return palettes[(level-1)%len(palettes)]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
