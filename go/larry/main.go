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

const (
	startMenu   = 0
	startScores = 1
)

// UTF-8 gameplay glyphs (escapes keep source encoding-safe)
const (
	glyphLarry = '\u2B22' // BLACK HEXAGON
	glyphRight = '\u25B6' // BLACK RIGHT-POINTING TRIANGLE
	glyphLeft  = '\u25C0' // BLACK LEFT-POINTING TRIANGLE
	glyphBlock = '\u2588' // FULL BLOCK
	glyphRail  = '\u2550' // BOX DRAWINGS DOUBLE HORIZONTAL
)


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
	startView       int // startMenu | startScores
	menuIndex       int // 0 Start, 1 High Scores, 2 Quit
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

	enableUTF8Console()

	s, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := s.Init(); err != nil {
		panic(err)
	}
	enableUTF8Console() // re-apply CP65001 + Unicode font after tcell init
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
	g.startView = startMenu
	g.menuIndex = 0
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
				if g.handleQuit(e) {
					return
				}
				if g.handleInput(e) {
					return
				}
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
					glyph = []rune{glyphRail, glyphRight}
				} else {
					glyph = []rune{glyphLeft, glyphRail}
				}
			case 1: // regular
				minSpd, maxSpd = 2, 4
				color = g.theme.carRegular
				glyph = []rune{glyphLeft, glyphBlock, glyphRight}
			default: // 2: semi
				minSpd, maxSpd = 1, 3
				color = g.theme.carSemi
				if dirRight {
					glyph = []rune{glyphBlock, glyphBlock, glyphBlock, glyphBlock, glyphRight}
				} else {
					glyph = []rune{glyphLeft, glyphBlock, glyphBlock, glyphBlock, glyphBlock}
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

func (g *game) handleInput(e *tcell.EventKey) bool {
	// Handle start screen
	if g.showStartScreen {
		return g.handleStartInput(e)
	}
	// ignore inputs for a brief period after death/gameover to prevent buffered arrows into name field
	if time.Now().Before(g.acceptInputAfter) {
		return false
	}
	if g.enteringName {
		// Simple name input handler
		switch e.Key() {
		case tcell.KeyEnter:
			g.commitScoreName()
			return false
		case tcell.KeyEscape:
			g.enteringName = false
			return false
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			return false
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(g.nameBuffer) > 0 {
				g.nameBuffer = g.nameBuffer[:len(g.nameBuffer)-1]
			}
			return false
		case tcell.KeyRune:
			r := e.Rune()
			if r >= 32 && r <= 126 && len(g.nameBuffer) < 8 {
				g.nameBuffer += string(r)
			}
			return false
		default:
			return false
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
		return false
	}
	if g.paused {
		return false
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
	return false
}

func (g *game) handleStartInput(e *tcell.EventKey) bool {
	if g.startView == startScores {
		switch e.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			g.startView = startMenu
			return false
		case tcell.KeyRune:
			if e.Rune() == ' ' {
				g.startView = startMenu
			}
		}
		return false
	}

	const menuCount = 3
	switch e.Key() {
	case tcell.KeyUp:
		g.menuIndex = (g.menuIndex - 1 + menuCount) % menuCount
		return false
	case tcell.KeyDown:
		g.menuIndex = (g.menuIndex + 1) % menuCount
		return false
	case tcell.KeyEnter:
		return g.activateMenuItem()
	case tcell.KeyRune:
		r := e.Rune()
		switch r {
		case ' ', '\n', '\r':
			return g.activateMenuItem()
		case 'w', 'W':
			g.menuIndex = (g.menuIndex - 1 + menuCount) % menuCount
		case 's', 'S':
			g.menuIndex = (g.menuIndex + 1) % menuCount
		case '1':
			g.menuIndex = 0
			return g.activateMenuItem()
		case 'h', 'H', '2':
			g.menuIndex = 1
			return g.activateMenuItem()
		case 'q', 'Q', '3':
			g.menuIndex = 2
			return g.activateMenuItem()
		}
	}
	return false
}

func (g *game) activateMenuItem() bool {
	switch g.menuIndex {
	case 0: // Start
		g.showStartScreen = false
		g.startView = startMenu
		return false
	case 1: // High Scores
		g.startView = startScores
		return false
	case 2: // Quit
		return true
	}
	return false
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

func (g *game) advanceLanes() {
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
}

func (g *game) update() {
	if g.showStartScreen {
		// Keep background traffic moving behind the menu
		g.advanceLanes()
		return
	}
	if g.paused {
		return
	}
	if g.enteringName {
		return
	}
	g.advanceLanes()

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

func (g *game) drawPlayfieldBackground() {
	w, h := g.width, g.height
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
			g.screen.SetContent(x, y, ' ', nil, st)
		}
	}
}

func (g *game) drawVehicles() {
	w, h := g.width, g.height
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
					g.screen.SetContent(x, ln.y, ch, nil, st)
				}
			}
		}
	}
}

func (g *game) render() {
	s := g.screen
	s.Clear()

	// Show start screen if active
	if g.showStartScreen {
		g.drawStartScreen()
		s.Show()
		return
	}

	w := g.width
	g.drawPlayfieldBackground()
	g.drawVehicles()

	// Draw HUD - will refresh only when score changes
	if g.score != g.lastRenderedScore {
		g.updateHUD()
		g.lastRenderedScore = g.score
	}
	// HUD uses Larry's contrasting color to clearly separate from playfield
	hudStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.theme.frog).Bold(true)
	drawText(s, 0, 0, spaces(w), hudStyle)
	drawText(s, 0, 0, g.hudLine, hudStyle)

	// Draw Larry as a filled hexagon (UTF-8)
	frogStyle := tcell.StyleDefault.Foreground(g.theme.frog).Bold(true)
	s.SetContent(g.frogX, g.frogY, glyphLarry, nil, frogStyle)

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
	g.startView = startMenu
	g.menuIndex = 0
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

func (g *game) handleQuit(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyCtrlC {
		return true
	}
	if e.Key() == tcell.KeyEscape {
		// Esc from high scores returns to the menu (handled in handleStartInput)
		if g.showStartScreen && g.startView == startScores {
			return false
		}
		return true
	}
	return false
}

// Box-drawing / section markers (single-width UTF-8)
const (
	boxTL       = '╔'
	boxTR       = '╗'
	boxBL       = '╚'
	boxBR       = '╝'
	boxH        = '═'
	boxV        = '║'
	boxML       = '╠' // middle tee left
	boxMR       = '╣' // middle tee right
	boxDivider  = "\x01" // sentinel: draw an ╠═══╣ section rule
)

func drawText(s tcell.Screen, x, y int, text string, st tcell.Style) {
	col := 0
	for _, ch := range text {
		s.SetContent(x+col, y, ch, nil, st)
		col++
	}
}

func drawCentered(s tcell.Screen, cx, cy int, text string, st tcell.Style) {
	x := cx - len([]rune(text))/2
	drawText(s, x, cy, text, st)
}

func boxHoriz(innerWidth int, left, right rune) string {
	var b strings.Builder
	b.Grow(innerWidth + 2)
	b.WriteRune(left)
	for i := 0; i < innerWidth; i++ {
		b.WriteRune(boxH)
	}
	b.WriteRune(right)
	return b.String()
}

// drawBorderedBox draws a UTF-8 box-drawing rectangle centered on cx.
// Pass boxDivider as a line to insert an ╠═══╣ section separator.
func drawBorderedBox(s tcell.Screen, cx, topY, innerWidth int, borderStyle, fillStyle tcell.Style, lines []string, lineStyles []tcell.Style) {
	w, h := s.Size()
	if innerWidth < 1 || topY >= h {
		return
	}
	boxW := innerWidth + 2
	left := cx - boxW/2
	if left < 0 {
		left = 0
	}
	if left+boxW > w {
		boxW = w - left
		innerWidth = boxW - 2
		if innerWidth < 1 {
			return
		}
	}
	drawText(s, left, topY, boxHoriz(innerWidth, boxTL, boxTR), borderStyle)
	for i, line := range lines {
		y := topY + 1 + i
		if y < 0 || y >= h {
			continue
		}
		if line == boxDivider {
			drawText(s, left, y, boxHoriz(innerWidth, boxML, boxMR), borderStyle)
			continue
		}
		rowStyle := fillStyle
		if i < len(lineStyles) {
			rowStyle = lineStyles[i]
		}
		// Fill inner row; keep vertical borders in borderStyle
		inner := spaces(innerWidth)
		drawText(s, left+1, y, inner, rowStyle)
		s.SetContent(left, y, boxV, nil, borderStyle)
		s.SetContent(left+boxW-1, y, boxV, nil, borderStyle)
		runes := []rune(line)
		if len(runes) > innerWidth {
			runes = runes[:innerWidth]
		}
		pad := (innerWidth - len(runes)) / 2
		drawText(s, left+1+pad, y, string(runes), rowStyle)
	}
	bottomY := topY + 1 + len(lines)
	if bottomY >= 0 && bottomY < h {
		drawText(s, left, bottomY, boxHoriz(innerWidth, boxBL, boxBR), borderStyle)
	}
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
	// Build the HUD string with UTF-8 separators
	w := g.width
	left := fmt.Sprintf("Score:%d  │  Level:%d  │  Lives:%d", g.score, g.level, g.lives)
	help := "  (Space:Pause Esc:Quit)"
	right := fmt.Sprintf("Top:%d  ·  Best:%d", g.topScore, g.historyTop)
	if len([]rune(left))+len(help)+len([]rune(right))+1 <= w {
		left += help
	}
	hudLine := left
	leftLen := len([]rune(left))
	rightLen := len([]rune(right))
	if leftLen+1+rightLen < w {
		pad := w - leftLen - rightLen
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
	boxBg := tcell.ColorBlack
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	fillStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg).Bold(true)
	inner := 16
	topY := h/2 - 2
	if topY < 0 {
		topY = 0
	}
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, fillStyle,
		[]string{"PAUSED", "Space to resume"},
		[]tcell.Style{fillStyle, tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg)})
}

func (g *game) drawNameEntryOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	prov := g.getProvisionalScores()
	maxScores := 10
	if h < 20 {
		maxScores = 3
	}
	name := g.nameBuffer
	if name == "" {
		name = "_"
	}
	lines := make([]string, 0, maxScores+5)
	styles := make([]tcell.Style, 0, maxScores+5)
	boxBg := tcell.ColorBlack
	titleSt := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(boxBg).Bold(true)
	bodySt := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	champSt := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow).Bold(true)
	promptSt := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg).Bold(true)
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)

	lines = append(lines, "NEW HIGH SCORE!")
	styles = append(styles, titleSt)
	lines = append(lines, boxDivider)
	styles = append(styles, bodySt)
	show := maxScores
	if show > len(prov) {
		show = len(prov)
	}
	for i := 0; i < show; i++ {
		e := prov[i]
		line := fmt.Sprintf("%2d. %-8s  %6d  %s", i+1, e.Name, e.Score, e.Date)
		lines = append(lines, line)
		if i == 0 {
			styles = append(styles, champSt)
		} else {
			styles = append(styles, bodySt)
		}
	}
	lines = append(lines, boxDivider)
	styles = append(styles, bodySt)
	lines = append(lines, "Enter Name: "+name)
	styles = append(styles, promptSt)

	inner := 32
	for _, ln := range lines {
		if ln == boxDivider {
			continue
		}
		if n := len([]rune(ln)) + 2; n > inner {
			inner = n
		}
	}
	topY := h/2 - (len(lines)+2)/2
	if topY < 0 {
		topY = 0
	}
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, bodySt, lines, styles)
}

func (g *game) drawScoreboardOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	boxBg := tcell.ColorBlack
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	bodySt := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	titleSt := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(boxBg).Bold(true)
	champSt := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow).Bold(true)

	lines := []string{"GAME OVER", boxDivider}
	styles := []tcell.Style{titleSt, bodySt}
	show := 10
	if show > len(g.highScores) {
		show = len(g.highScores)
	}
	for i := 0; i < show; i++ {
		e := g.highScores[i]
		lines = append(lines, fmt.Sprintf("%2d. %-8s  %6d  %s", i+1, e.Name, e.Score, e.Date))
		if i == 0 {
			styles = append(styles, champSt)
		} else {
			styles = append(styles, bodySt)
		}
	}
	lines = append(lines, boxDivider)
	styles = append(styles, bodySt)
	lines = append(lines, fmt.Sprintf("Your Score: %d", g.score))
	styles = append(styles, tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg).Bold(true))

	inner := 32
	for _, ln := range lines {
		if ln == boxDivider {
			continue
		}
		if n := len([]rune(ln)) + 2; n > inner {
			inner = n
		}
	}
	topY := h/2 - (len(lines)+2)/2
	if topY < 0 {
		topY = 0
	}
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, bodySt, lines, styles)
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
	// Letterforms only — outer UTF-8 box is drawn by drawStartScreen
	return []string{
		"L     AAA  RRRR  RRRR  Y   Y",
		"L    A   A R   R R   R  Y Y ",
		"L    AAAAA RRRR  RRRR    Y  ",
		"L    A   A R  R  R  R    Y  ",
		"LLLL A   A R   R R   R   Y  ",
	}
}

func (g *game) drawStartScreen() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}

	// Animated playfield traffic behind the menu
	g.drawPlayfieldBackground()
	g.drawVehicles()

	if g.startView == startScores {
		g.drawStartHighScores()
		return
	}

	boxBg := tcell.ColorBlack
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	fillStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	titleStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	subStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	champStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(boxBg).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	selectedStyle := tcell.StyleDefault.Background(g.theme.frog).Foreground(tcell.ColorBlack).Bold(true)
	helpStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg).Bold(true)
	hintStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)

	var highScoreText string
	if len(g.highScores) > 0 {
		topScore := g.highScores[0]
		highScoreText = fmt.Sprintf("Champion: %d by %s (%s)", topScore.Score, topScore.Name, topScore.Date)
	} else {
		highScoreText = "Champion: —"
	}

	ascii := getLarryASCII()
	blinkOn := (time.Now().UnixNano()/int64(time.Millisecond)/400)%2 == 0
	menuItems := []string{"Start", "High Scores", "Quit"}

	lines := make([]string, 0, 20)
	styles := make([]tcell.Style, 0, 20)
	for _, line := range ascii {
		lines = append(lines, line)
		styles = append(styles, titleStyle)
	}
	lines = append(lines, "")
	styles = append(styles, fillStyle)
	lines = append(lines, "Terminal Frogger")
	styles = append(styles, subStyle)
	lines = append(lines, highScoreText)
	styles = append(styles, champStyle)
	lines = append(lines, boxDivider)
	styles = append(styles, fillStyle)
	for i, label := range menuItems {
		cursor := "  "
		st := normalStyle
		if i == g.menuIndex {
			st = selectedStyle
			if blinkOn {
				cursor = "→ "
			}
		}
		lines = append(lines, cursor+label)
		styles = append(styles, st)
	}
	lines = append(lines, boxDivider)
	styles = append(styles, fillStyle)
	lines = append(lines, "↑↓ Select  ·  Enter Confirm")
	styles = append(styles, helpStyle)
	lines = append(lines, "Arrows or WASD to move in game")
	styles = append(styles, hintStyle)

	inner := 34
	for _, ln := range lines {
		if ln == boxDivider {
			continue
		}
		if n := len([]rune(ln)) + 2; n > inner {
			inner = n
		}
	}
	if inner > w-2 {
		inner = max(1, w-2)
	}
	topY := h/2 - (len(lines)+2)/2
	if topY < 1 {
		topY = 1
	}
	// Menu drawn last so it stays above moving traffic
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, fillStyle, lines, styles)
}

func (g *game) drawStartHighScores() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	boxBg := tcell.ColorBlack
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	fillStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(boxBg).Bold(true)
	champStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow).Bold(true)
	hintStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg).Bold(true)

	lines := []string{"HIGH SCORES", boxDivider}
	styles := []tcell.Style{titleStyle, fillStyle}
	if len(g.highScores) == 0 {
		lines = append(lines, "No scores yet — be the first!")
		styles = append(styles, fillStyle)
	} else {
		show := 10
		if show > len(g.highScores) {
			show = len(g.highScores)
		}
		for i := 0; i < show; i++ {
			e := g.highScores[i]
			lines = append(lines, fmt.Sprintf("%2d. %-8s  %6d  %s", i+1, e.Name, e.Score, e.Date))
			if i == 0 {
				styles = append(styles, champStyle)
			} else {
				styles = append(styles, fillStyle)
			}
		}
	}
	lines = append(lines, boxDivider)
	styles = append(styles, fillStyle)
	lines = append(lines, "Esc or Enter to return")
	styles = append(styles, hintStyle)

	inner := 32
	for _, ln := range lines {
		if ln == boxDivider {
			continue
		}
		if n := len([]rune(ln)) + 2; n > inner {
			inner = n
		}
	}
	if inner > w-2 {
		inner = max(1, w-2)
	}
	topY := h/2 - (len(lines)+2)/2
	if topY < 1 {
		topY = 1
	}
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, fillStyle, lines, styles)
}
