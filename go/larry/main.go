package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	level            int
	score            int
	topScore         int
	lives            int
	frogX            int
	frogY            int
	highestY         int
	hudY             int
	lanes            []lane
	safeTopY         int
	safeBottomY      int
	safeRow          []bool
	rng              *rand.Rand
	theme            theme
	paused           bool
	events           chan tcell.Event
	acceptInputAfter time.Time
	// Per-level score decay
	scoreTimerActive   bool
	nextScoreDecrement time.Time
	// HUD throttling
	hudLine           string
	lastRenderedScore int
	// High scores
	highScores   []scoreEntry
	historyTop   int
	gameOver     bool
	enteringName bool
	nameBuffer   string
	// Start screen
	showStartScreen bool
}

type scoreEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Time  int64  `json:"time"`
	Date  string `json:"date,omitempty"`
}

func main() {
	// Set up panic recovery to ensure cleanup
	defer func() {
		if r := recover(); r != nil {
			// Reset terminal colors to default using ANSI escape codes
			fmt.Print("\033[0m")
			// Also reset cursor visibility
			fmt.Print("\033[?25h")
			panic(r) // Re-panic after cleanup
		}
	}()

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

	// Set up signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Cleanup function to reset terminal colors
	cleanup := func() {
		// Reset terminal colors to default using ANSI escape codes
		fmt.Print("\033[0m")
		// Also reset cursor visibility
		fmt.Print("\033[?25h")
		// Finalize the screen
		s.Fini()
	}

	// Ensure cleanup runs on exit
	defer cleanup()

	setTerminalTitle("Go Larry!")

	g := &game{screen: s, rng: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))}
	g.loadHighScores()
	if len(g.highScores) > 0 {
		g.historyTop = g.highScores[0].Score
	}
	g.showStartScreen = true
	g.initLevel(1)

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			events <- s.PollEvent()
		}
	}()
	g.events = events

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
		case <-sigChan:
			// Handle Ctrl+C and other termination signals
			return
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

func (g *game) respawnAtStart() {
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
}

func (g *game) initLevel(level int) {
	g.level = level
	g.width, g.height = g.screen.Size()
	// Lives/score are set on first game start; keep values across levels.
	if g.lives <= 0 {
		g.lives = 3
		g.score = 0
	}
	g.lastRenderedScore = -1 // force initial HUD draw
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.theme = themeForLevel(level)
	// score decay starts only after first action each level
	g.scoreTimerActive = false
	g.updateHUD()
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
	// Clear input buffer and pause input to prevent instant death on new level
	g.flushInput()
	g.acceptInputAfter = time.Now().Add(200 * time.Millisecond)
	// Reward: extra life each cleared level
	g.lives++
	g.theme = themeForLevel(g.level)
	// reset decay timer for new level
	g.scoreTimerActive = false
	g.updateHUD()
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
		var densityFactor, speedFactor float64
		if g.level <= 5 {
			// Original progression for levels 1-5
			densityFactor = 0.5 + 0.05*float64(max(0, g.level-1)) // 0.5 at L1, +5% each level
			speedFactor = 0.67 + 0.05*float64(max(0, g.level-1))  // ~33% slower at L1, +5% each level
		} else {
			// New progression after level 5
			// Speed increases each level after 5
			speedFactor = 0.92 + 0.08*float64(g.level-5) // Start at 0.92, +8% each level after 5
			// Density only increases every 5 levels after level 5 (at levels 10, 15, 20, etc.)
			densityIncreases := (g.level - 5) / 5
			densityFactor = 0.75 + 0.1*float64(densityIncreases) // Start at 0.75, +10% every 5 levels
		}

		// Apply caps
		if densityFactor > 2.0 {
			densityFactor = 2.0
		}
		if speedFactor > 2.0 {
			speedFactor = 2.0
		}

		for li := 0; li < lanesThisRoad && y < h-1; li++ {
			// Vehicle class selection per lane
			vehType := g.rng.IntN(3) // 0 compact, 1 regular, 2 semi
			var minSpd, maxSpd int
			var color tcell.Color
			var glyph []rune
			switch vehType {
			case 0: // compact
				minSpd, maxSpd = 3, 5
				color = g.theme.carSmall
				if dirRight {
					glyph = []rune{'=', '>'} // carSmall '=>'
				} else {
					glyph = []rune{'<', '='} // carSmall '<='
				}
			case 1: // regular
				minSpd, maxSpd = 2, 4
				color = g.theme.carRegular
				// visually symmetric
				glyph = []rune{'<', '#', '>'}
			default: // 2: semi
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
	// Handle start screen
	if g.showStartScreen {
		// Any key press starts the game
		g.showStartScreen = false
		return
	}
	// ignore inputs for a brief period after death/gameover to prevent buffered arrows into name field
	if time.Now().Before(g.acceptInputAfter) {
		return
	}
	if g.enteringName {
		// Simple name input handler
		switch e.Key() {
		case tcell.KeyEnter:
			g.commitScoreName()
			return
		case tcell.KeyEscape:
			g.enteringName = false
			return
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			return
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(g.nameBuffer) > 0 {
				g.nameBuffer = g.nameBuffer[:len(g.nameBuffer)-1]
			}
			return
		case tcell.KeyRune:
			r := e.Rune()
			if r >= 32 && r <= 126 && len(g.nameBuffer) < 8 {
				g.nameBuffer += string(r)
			}
			return
		default:
			return
		}
	}
	// Toggle pause on Space
	if e.Key() == tcell.KeyRune && e.Rune() == ' ' {
		if g.paused {
			// resuming
			g.paused = false
			if g.scoreTimerActive {
				g.nextScoreDecrement = time.Now().Add(time.Second)
			}
		} else {
			// pausing
			g.paused = true
		}
		return
	}
	if g.paused {
		return
	}
	moved := false
	switch e.Key() {
	case tcell.KeyLeft:
		g.frogX--
		moved = true
	case tcell.KeyRight:
		g.frogX++
		moved = true
	case tcell.KeyUp:
		g.frogY--
		moved = true
		if g.frogY < g.highestY {
			g.score += (g.highestY - g.frogY) * 10 // per-line bonus when advancing upward
			g.highestY = g.frogY
			if g.score > g.topScore {
				g.topScore = g.score
			}
		}
	case tcell.KeyDown:
		g.frogY++
		moved = true
	default:
		switch e.Rune() {
		case 'a', 'A':
			g.frogX--
			moved = true
		case 'd', 'D':
			g.frogX++
			moved = true
		case 'w', 'W':
			g.frogY--
			moved = true
			if g.frogY < g.highestY {
				g.score += (g.highestY - g.frogY) * 10
				g.highestY = g.frogY
				if g.score > g.topScore {
					g.topScore = g.score
				}
			}
		case 's', 'S':
			g.frogY++
			moved = true
		}
	}
	g.clampFrog()
	if moved && !g.scoreTimerActive {
		g.scoreTimerActive = true
		g.nextScoreDecrement = time.Now().Add(time.Second)
	}
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
	if g.enteringName {
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
							// Delay accepting input until overlay is up
							g.acceptInputAfter = time.Now().Add(1250 * time.Millisecond) // 1050ms flash + 200ms buffer
							g.gameOverSequence()
						} else {
							// Respawn at start row and show brief message
							g.respawnAtStart()
							// Drain any pending input before showing overlay
							g.flushInput()
							g.acceptInputAfter = time.Now().Add(900 * time.Millisecond) // 700ms flash + 200ms buffer
							g.youDiedFlash()
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

	// Per-second score decay while level is active
	if g.scoreTimerActive && time.Now().After(g.nextScoreDecrement) {
		if g.score > 0 {
			g.score--
		}
		g.nextScoreDecrement = time.Now().Add(time.Second)
	}
}

func (g *game) render() {
	s := g.screen
	s.Clear()
	w, h := g.width, g.height

	// Show start screen if active
	if g.showStartScreen {
		g.drawStartScreen()
		s.Show()
		return
	}

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

	// Draw HUD - will refresh only when score changes
	if g.score != g.lastRenderedScore {
		g.updateHUD()
		g.lastRenderedScore = g.score
	}
	// HUD uses Larry's contrasting color to clearly separate from playfield
	hudStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.theme.frog).Bold(true)
	drawText(s, 0, 0, spaces(w), hudStyle)
	drawText(s, 0, 0, g.hudLine, hudStyle)

	// Draw Larry as a green '@' for wide-compat terminals
	frogStyle := tcell.StyleDefault.Foreground(g.theme.frog).Bold(true)
	s.SetContent(g.frogX, g.frogY, '@', nil, frogStyle)

	// Ensure overlays are drawn last, on top of vehicles and frog
	if g.enteringName {
		g.drawNameEntryOverlay()
	} else if g.gameOver {
		g.drawScoreboardOverlay()
	} else if g.paused {
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
		time.Sleep(350 * time.Millisecond)
	}
}

func (g *game) gameOverSequence() {
	g.gameOverFlash()
	g.gameOver = true
	// Check if score qualifies for top 10
	qualifies := false
	if len(g.highScores) < 10 {
		qualifies = g.score > 0
	} else if g.score > g.highScores[len(g.highScores)-1].Score {
		qualifies = true
	}
	if qualifies {
		g.enteringName = true
		g.nameBuffer = ""
		return
	}
	g.resetGame()
}

func (g *game) commitScoreName() {
	name := strings.TrimSpace(g.nameBuffer)
	if name == "" {
		name = "PLAYER"
	}
	if len(name) > 8 {
		name = name[:8]
	}
	now := time.Now()
	entry := scoreEntry{Name: name, Score: g.score, Time: now.Unix(), Date: now.Format("010206")}
	g.highScores = append(g.highScores, entry)
	// sort desc
	for i := 0; i < len(g.highScores); i++ {
		for j := i + 1; j < len(g.highScores); j++ {
			if g.highScores[j].Score > g.highScores[i].Score {
				g.highScores[i], g.highScores[j] = g.highScores[j], g.highScores[i]
			}
		}
	}
	if len(g.highScores) > 10 {
		g.highScores = g.highScores[:10]
	}
	g.saveHighScores()
	if len(g.highScores) > 0 {
		g.historyTop = g.highScores[0].Score
	}
	g.enteringName = false
	g.resetGame()
}

func (g *game) resetGame() {
	g.lives = 3
	g.score = 0
	g.lastRenderedScore = -1
	g.level = 1
	g.theme = themeForLevel(g.level)
	g.createLanes()
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.gameOver = false
	g.showStartScreen = true
	g.acceptInputAfter = time.Now().Add(200 * time.Millisecond)
	// fresh start: no decay until first move
	g.scoreTimerActive = false
	g.updateHUD()
}

func (g *game) loadHighScores() {
	data, err := os.ReadFile("larry.scores.json")
	if err != nil {
		return
	}
	var list []scoreEntry
	if json.Unmarshal(data, &list) == nil {
		g.highScores = list
	}
}

func (g *game) saveHighScores() {
	data, err := json.MarshalIndent(g.highScores, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile("larry.scores.json", data, 0644)
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
		time.Sleep(350 * time.Millisecond)
	}
}

func (g *game) flushInput() {
	if g.events == nil {
		return
	}
	for {
		select {
		case <-g.events:
			// drop
		default:
			return
		}
	}
}

func handleQuit(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlC {
		return true
	}
	return false
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

func (g *game) updateHUD() {
	// Build the HUD string
	w := g.width
	left := fmt.Sprintf("Score:%d  Level:%d  Lives:%d", g.score, g.level, g.lives)
	help := "  (Space:Pause Esc:Quit)"
	right := fmt.Sprintf("Top:%d  Best:%d", g.topScore, g.historyTop)
	if len(left)+len(help)+len(right)+1 <= w {
		left += help
	}
	hudLine := left
	if len(left)+1+len(right) < w {
		pad := w - len(left) - len(right)
		if pad < 1 {
			pad = 1
		}
		hudLine = left + spaces(pad) + right
	}
	g.hudLine = hudLine
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

func (g *game) drawNameEntryOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	title := "NEW HIGH SCORE!"
	// Reserve space for title + scores + prompt (up to 15 lines total)
	y0 := h/2 - 7
	if y0 < 0 {
		y0 = 0
	}
	if y0+15 >= h {
		y0 = max(0, h-16)
	}
	st := tcell.StyleDefault.Background(g.theme.frog).Foreground(tcell.ColorBlack).Bold(true)
	for dy := 0; dy < 16; dy++ {
		drawText(g.screen, 0, y0+dy, spaces(w), st)
	}
	drawCentered(g.screen, w/2, y0+1, title, st)
	prov := g.getProvisionalScores()
	// Show top 10 if space allows, otherwise top 3
	maxScores := 10
	if y0+3+maxScores+4 >= h { // title + scores + gap + prompt + cursor
		maxScores = 3
	}
	g.drawHighScoreListAt(w/2, y0+3, st, prov, maxScores)
	// Prompt for name below the score list
	promptY := y0 + 3 + maxScores + 1
	promptText := "Enter Name: "
	name := g.nameBuffer
	if name == "" {
		name = "_"
	}
	drawCentered(g.screen, w/2, promptY, promptText+name, st)
}

func (g *game) drawScoreboardOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	title := "GAME OVER"
	y0 := h/2 - 6
	if y0 < 0 {
		y0 = 0
	}
	if y0+12 >= h {
		y0 = max(0, h-13)
	}
	st := tcell.StyleDefault.Background(g.theme.frog).Foreground(tcell.ColorBlack).Bold(true)
	for dy := 0; dy < 13; dy++ {
		drawText(g.screen, 0, y0+dy, spaces(w), st)
	}
	drawCentered(g.screen, w/2, y0+1, title, st)
	g.drawHighScoreListAt(w/2, y0+3, st, g.highScores, 10)
	// If player didn't make Top 10, show their score/name in the prompt area
	if len(g.highScores) == 0 || g.score > g.highScores[len(g.highScores)-1].Score {
		// reached only when no scores; otherwise name entry covers this path
		// fallback to simple retry prompt
		drawCentered(g.screen, w/2, y0+11, "Hit Return to Try Again", st)
	} else {
		you := fmt.Sprintf("Your Score: %d", g.score)
		drawCentered(g.screen, w/2, y0+11, you, st)
	}
}

func (g *game) drawHighScoreListAt(cx, startY int, st tcell.Style, list []scoreEntry, maxScores int) {
	// Render up to maxScores entries with the top entry highlighted
	for i := 0; i < maxScores && i < len(list); i++ {
		e := list[i]
		// Include date in MMDDYY
		line := fmt.Sprintf("%2d. %-8s  %6d  %s", i+1, e.Name, e.Score, e.Date)
		rowStyle := st
		if i == 0 {
			// Highlight champion
			rowStyle = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack).Bold(true)
		}
		drawCentered(g.screen, cx, startY+i, line, rowStyle)
	}
}

func (g *game) getProvisionalScores() []scoreEntry {
	list := make([]scoreEntry, len(g.highScores))
	copy(list, g.highScores)
	now := time.Now()
	list = append(list, scoreEntry{Name: "YOUR SCORE", Score: g.score, Time: now.Unix(), Date: now.Format("010206")})
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Score > list[i].Score {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if len(list) > 10 {
		list = list[:10]
	}
	return list
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

func getLarryASCII() []string {
	return []string{
		"+------------------------------+",
		"| L     AAA  RRRR  RRRR  Y   Y |",
		"| L    A   A R   R R   R  Y Y  |",
		"| L    AAAAA RRRR  RRRR    Y   |",
		"| L    A   A R  R  R  R    Y   |",
		"| LLLL A   A R   R R   R   Y   |",
		"+------------------------------+",
	}
}

func (g *game) drawStartScreen() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}

	// Fill background with a nice gradient-like pattern
	for y := 0; y < h; y++ {
		var bg tcell.Color
		switch y % 3 {
		case 0:
			bg = g.theme.road
		case 1:
			bg = g.theme.river
		default:
			bg = g.theme.safe
		}
		st := tcell.StyleDefault.Background(bg)
		for x := 0; x < w; x++ {
			g.screen.SetContent(x, y, ' ', nil, st)
		}
	}

	// Get ASCII art
	ascii := getLarryASCII()
	asciiHeight := len(ascii)
	startY := h/2 - asciiHeight/2 - 3

	// Draw ASCII art
	titleStyle := tcell.StyleDefault.Foreground(g.theme.frog).Bold(true)
	for i, line := range ascii {
		y := startY + i
		if y >= 0 && y < h {
			drawCentered(g.screen, w/2, y, line, titleStyle)
		}
	}

	// Draw high score with player name and date
	highScoreY := startY + asciiHeight + 2
	if highScoreY >= 0 && highScoreY < h {
		var highScoreText string
		if len(g.highScores) > 0 {
			topScore := g.highScores[0]
			highScoreText = fmt.Sprintf("High Score: %d by %s (%s)", topScore.Score, topScore.Name, topScore.Date)
		} else {
			highScoreText = "High Score: 0"
		}
		scoreStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
		drawCentered(g.screen, w/2, highScoreY, highScoreText, scoreStyle)
	}

	// Draw start prompt
	promptY := highScoreY + 3
	if promptY >= 0 && promptY < h {
		promptText := "Press any key to start"
		promptStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		drawCentered(g.screen, w/2, promptY, promptText, promptStyle)
	}

	// Draw controls help
	helpY := promptY + 2
	if helpY >= 0 && helpY < h {
		helpText := "Use arrow keys or WASD to move"
		helpStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		drawCentered(g.screen, w/2, helpY, helpText, helpStyle)
	}
}
