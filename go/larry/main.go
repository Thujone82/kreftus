package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
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
	glyphLarry   = '\u2B22' // BLACK HEXAGON
	glyphRight   = '\u25B6' // BLACK RIGHT-POINTING TRIANGLE
	glyphLeft    = '\u25C0' // BLACK LEFT-POINTING TRIANGLE
	glyphBlock   = '\u2588' // FULL BLOCK
	glyphRail    = '\u2550' // BOX DRAWINGS DOUBLE HORIZONTAL
	glyphTruck   = '\u2593' // ▓ dark shade — truck box
	glyphBike    = '\u25A9' // ▩ crosshatch — motorcycle body
	glyphCarBox  = '\u25D9' // ◙ inverse circle — car box
	glyphGoalA   = '\u259A' // ▚ goal checker
	glyphGoalB   = '\u259E' // ▞ goal checker
	glyphDebris  = '\u2619' // ☙ impassable safe-lane debris
	glyphHeart   = '\u2665' // ♥ life pickup (L8+)
	glyphDiamond = '\u2666' // ♦ score pickup (L10+)
)

// larryHighlight is the goal-checker yellow — always distinct from vehicle hues.
const larryHighlight = tcell.ColorYellow

// larryVersion is written into high-score entries (bump with releases).
const larryVersion = "1.5"

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
	debris           [][]bool // impassable cells on mid safe lanes (L6+)
	hasHeart         bool // L8+: one ♥ on a mid safe lane
	heartX           int
	heartY           int
	hasDiamond       bool // L10+: one ♦ on the top mid safe lane
	diamondX         int
	diamondY         int
	heartsCollected  int // session ♥ pickups (saved on high score)
	gemsCollected    int // session ♦ pickups (saved on high score)
	hudNoticeUntil   time.Time // inverted HUD pickup flash
	hudNoticeText    string    // e.g. "+1 Life" / "+1,000 Points"
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
	scoreIndex      int // selected row on High Scores view
	confirmMenu     bool // Esc confirm: return to main menu?
	confirmYes      bool // true = Larry on Yes; false = Larry on No (default)
	testMode        bool // -testlvl: skip score file writes
}

type scoreEntry struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Time    int64  `json:"time"`
	Date    string `json:"date,omitempty"`
	Level   int    `json:"level"`             // level at death (0 = pre-1.5 / unknown)
	Hearts  int    `json:"hearts"`            // ♥ collected that run
	Gems    int    `json:"gems"`              // ♦ collected that run
	Version string `json:"version,omitempty"` // larry version that wrote the entry
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
	resizeToPreferred()

	s, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := s.Init(); err != nil {
		panic(err)
	}
	enableUTF8Console() // re-apply CP65001 + Unicode font after tcell init
	resizeToPreferred()
	s.Sync()
	// Avoid mouse/paste floods filling the event queue while the menu sits idle
	s.DisableMouse()
	s.DisablePaste()
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
	if testLvl := parseTestLevel(); testLvl > 0 {
		g.beginTestLevel(testLvl)
	} else {
		g.initLevel(1)
	}

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return // screen finalized
			}
			// Only forward gameplay-relevant events. Never block on send: a full
			// channel must not stall PollEvent (that backs up tcell after idle/wake).
			switch ev.(type) {
			case *tcell.EventKey, *tcell.EventResize:
				select {
				case events <- ev:
				default:
					// drop; keep draining the console input queue
				}
			default:
				// discard mouse/OS noise
			}
		}
	}()
	g.events = events

	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()

	lastFrame := time.Now()
	lastInput := time.Now()
	lastIdleDraw := time.Time{}

	handleEv := func(ev tcell.Event) bool {
		switch e := ev.(type) {
		case *tcell.EventResize:
			g.resize(e)
		case *tcell.EventKey:
			lastInput = time.Now()
			if g.handleQuit(e) {
				return true
			}
			if g.handleInput(e) {
				return true
			}
		}
		return false
	}

	for {
		select {
		case ev := <-events:
			if handleEv(ev) {
				return
			}
		case <-tick.C:
			// Drain any input that arrived with this frame so keys don't backlog
			for draining := true; draining; {
				select {
				case ev := <-events:
					if handleEv(ev) {
						return
					}
				default:
					draining = false
				}
			}
			now := time.Now()
			// After sleep/lock or a long stall, resync console and drop stale keys
			if now.Sub(lastFrame) > 2*time.Second {
				g.screen.Sync()
				g.flushInput()
			}
			lastFrame = now

			g.update()

			// Pause / confirm overlays are static — redrawing for hours thrashes the console
			if g.paused || g.confirmMenu {
				continue
			}
			// Idle start menu: keep traffic alive but throttle draws to ease console load
			if g.showStartScreen && now.Sub(lastInput) > 5*time.Second {
				if !lastIdleDraw.IsZero() && now.Sub(lastIdleDraw) < 500*time.Millisecond {
					continue
				}
				lastIdleDraw = now
			}
			g.render()
		case <-sigChan:
			// Handle Ctrl+C and other termination signals
			return
		}
	}
}

func (g *game) resize(e *tcell.EventResize) {
	// Sync remaps the console buffer to the new size; without it the right/bottom
	// edge keeps the old dimensions and stops updating after a resize.
	g.screen.Sync()
	w, h := 0, 0
	if e != nil {
		w, h = e.Size()
	}
	if w <= 0 || h <= 0 {
		w, h = g.screen.Size()
	}
	if w <= 0 || h <= 0 {
		return
	}
	g.width, g.height = w, h
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	// Respawn Larry to bottom safe shoulder and re-center horizontally
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.lastRenderedScore = -1 // force HUD redraw at new width
	g.createLanes()
	g.updateHUD()
	g.render()
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

// beginTestLevel starts gameplay at level as if prior levels were cleared.
func (g *game) beginTestLevel(level int) {
	if level < 1 {
		level = 1
	}
	g.testMode = true
	g.width, g.height = g.screen.Size()
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	rows := g.safeBottomY - g.safeTopY
	if rows < 1 {
		rows = 1
	}
	// Simulate clears: climb bonus + clear bonus − 10 time-decay per prior level
	score, lives := 0, 3
	for L := 1; L < level; L++ {
		score += rows * climbPointsPerRow(L)
		score += 100 * L
		score -= 10
		lives++
	}
	if score < 0 {
		score = 0
	}
	g.lives = lives
	g.score = score
	g.topScore = score
	g.heartsCollected = 0
	g.gemsCollected = 0
	g.showStartScreen = false
	g.startView = startMenu
	g.menuIndex = 0
	g.gameOver = false
	g.paused = false
	g.confirmMenu = false
	g.initLevel(level)
	g.flushInput()
	g.acceptInputAfter = time.Now().Add(300 * time.Millisecond)
}

// parseTestLevel reads undocumented -testlvl INT (or -testlvl=INT). Returns 0 if absent.
func parseTestLevel() int {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-testlvl" {
			if i+1 >= len(args) {
				return 0
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 {
				return 0
			}
			return n
		}
		if strings.HasPrefix(a, "-testlvl=") {
			n, err := strconv.Atoi(strings.TrimPrefix(a, "-testlvl="))
			if err != nil || n < 1 {
				return 0
			}
			return n
		}
	}
	return 0
}

func (g *game) nextLevel() {
	g.level++
	// Keep score/lives, reposition frog
	g.width, g.height = g.screen.Size()
	g.hudY = 0
	g.safeTopY = 1
	g.safeBottomY = g.height - 1
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	// Drop any in-flight moves before lane gen so held keys can't ride the transition
	g.flushInput()
	// Reward: extra life each cleared level
	g.lives++
	g.theme = themeForLevel(g.level)
	// reset decay timer for new level
	g.scoreTimerActive = false
	g.updateHUD()
	g.createLanes()
	// Flush again after setup (lag during createLanes can refill the channel), then
	// hold input briefly so nothing hops until the new level is on screen.
	g.flushInput()
	g.acceptInputAfter = time.Now().Add(300 * time.Millisecond)
}

func (g *game) createLanes() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	g.lanes = g.lanes[:0]
	g.safeRow = make([]bool, h)
	// Top/bottom playfield shoulders are always safe (HUD at row 0 is not playable)
	if g.safeTopY >= 0 && g.safeTopY < h {
		g.safeRow[g.safeTopY] = true
	}
	if g.safeBottomY >= 0 && g.safeBottomY < h {
		g.safeRow[g.safeBottomY] = true
	}
	// Generate roads: packs of lanes in one direction, then a safe gap, then flip.
	// Playfield between safeTopY and safeBottomY; HUD is at row 0.
	y := g.safeTopY + 1
	dirRight := g.rng.IntN(2) == 0
	for y < h-1 {
		// Road pack size grows slowly with level (was 4–6 from the start)
		minLanes, maxLanes := 3, 4
		if g.level >= 5 {
			minLanes, maxLanes = 3, 5
		}
		if g.level >= 10 {
			minLanes, maxLanes = 4, 6
		}
		lanesThisRoad := minLanes + g.rng.IntN(maxLanes-minLanes+1)
		if lanesThisRoad > 8 {
			lanesThisRoad = 8
		}

		// Difficulty curve: L1 easy; former L6 intensity arrives around L10
		var densityFactor, speedFactor float64
		if g.level <= 10 {
			// L1 density ~0.36 (~10% easier than 0.4); L10 ~0.75 (old L6 density)
			densityFactor = 0.36 + 0.043*float64(g.level-1)
			// L1 speed ~0.60; L10 ~0.92 (old L6 speed)
			speedFactor = 0.60 + 0.035*float64(g.level-1)
		} else {
			densityFactor = 0.75 + 0.05*float64(g.level-10)
			speedFactor = 0.92 + 0.04*float64(g.level-10)
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
			vehType := g.rng.IntN(3) // 0 motorcycle/compact, 1 car, 2 truck/semi
			var minSpd, maxSpd int
			var color tcell.Color
			var glyph []rune
			switch vehType {
			case 0: // motorcycle / compact — mix rail and ▩ bodies
				minSpd, maxSpd = 3, 5
				color = g.theme.carSmall
				body := glyphRail
				if g.rng.IntN(2) == 0 {
					body = glyphBike
				}
				if dirRight {
					glyph = []rune{body, glyphRight}
				} else {
					glyph = []rune{glyphLeft, body}
				}
			case 1: // car — mix █ and ◙ bodies
				minSpd, maxSpd = 2, 4
				color = g.theme.carRegular
				body := glyphBlock
				if g.rng.IntN(2) == 0 {
					body = glyphCarBox
				}
				glyph = []rune{glyphLeft, body, glyphRight}
			default: // 2: truck/semi — mix █ and ▓ boxes
				minSpd, maxSpd = 1, 3
				color = g.theme.carSemi
				box := glyphBlock
				if g.rng.IntN(2) == 0 {
					box = glyphTruck
				}
				if dirRight {
					glyph = []rune{box, box, box, box, glyphRight}
				} else {
					glyph = []rune{glyphLeft, box, box, box, box}
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
			// Circular pack: car starts must be at least (length+gap) apart around the ring
			minSpacing := length + baseGap
			if minSpacing < 1 {
				minSpacing = 1
			}
			num := w / minSpacing
			if num < 1 {
				num = 1
			}
			positions := make([]int, 0, num)
			if num == 1 {
				positions = append(positions, g.rng.IntN(max(1, w)))
			} else {
				// Equal strides around the torus (remainder spread on early gaps)
				stride := w / num
				rem := w % num
				pos := g.rng.IntN(max(1, w))
				for k := 0; k < num; k++ {
					positions = append(positions, pos%w)
					step := stride
					if k < rem {
						step++
					}
					pos += step
				}
			}
			g.lanes = append(g.lanes, lane{y: y, speedTicks: speed, dirRight: dirRight, cars: positions, width: w, length: length, glyph: glyph, color: color})
			if y >= 0 && y < h {
				g.safeRow[y] = false
			}
			y++
		}
		// Safe gaps: wider early, tighten toward late game
		gapMin, gapSpan := 2, 3 // 2..4
		if g.level >= 10 {
			gapMin, gapSpan = 1, 3 // 1..3
		}
		gap := gapMin + g.rng.IntN(gapSpan)
		for gi := 0; gi < gap && y < g.safeBottomY; gi++ {
			if y >= 0 && y < h {
				g.safeRow[y] = true
			}
			y++
		}
		// Flip road direction
		dirRight = !dirRight
	}
	g.placeDebris()
	g.placeHeart()
	g.placeDiamond()
}

func (g *game) placeDebris() {
	w, h := g.width, g.height
	g.debris = make([][]bool, h)
	for y := 0; y < h; y++ {
		g.debris[y] = make([]bool, w)
	}
	if g.level < 6 || w <= 0 || h <= 0 {
		return
	}
	chance := debrisChancePercent(g.level) / 100.0
	for y := 0; y < h; y++ {
		if y == g.safeTopY || y == g.safeBottomY {
			continue
		}
		if y >= len(g.safeRow) || !g.safeRow[y] {
			continue
		}
		for x := 0; x < w; x++ {
			if g.rng.Float64() < chance {
				g.debris[y][x] = true
			}
		}
	}
}

// debrisChancePercent: L6–10 add 1%/level (1…5%), then +0.5%/level, capped at 10%.
func debrisChancePercent(level int) float64 {
	if level < 6 {
		return 0
	}
	if level <= 10 {
		return float64(level - 5) // 1,2,3,4,5
	}
	pct := 5.0 + 0.5*float64(level-10)
	if pct > 10 {
		return 10
	}
	return pct
}

// climbPointsPerRow is +10 through level 10; from level 11 the level number is the per-row bonus.
func climbPointsPerRow(level int) int {
	if level >= 11 {
		return level
	}
	return 10
}

// placeHeart puts one ♥ at the center of the mid-playfield safe lane (L8+).
func (g *game) placeHeart() {
	g.hasHeart = false
	w, h := g.width, g.height
	if g.level < 8 || w <= 0 || h <= 0 {
		return
	}
	mid := (g.safeTopY + g.safeBottomY) / 2
	bestY, bestDist := -1, h+1
	for y := 0; y < h; y++ {
		if y == g.safeTopY || y == g.safeBottomY {
			continue
		}
		if y >= len(g.safeRow) || !g.safeRow[y] {
			continue
		}
		d := y - mid
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			bestY = y
		}
	}
	if bestY < 0 {
		return
	}
	x := w / 2
	g.heartX, g.heartY = x, bestY
	g.hasHeart = true
	// Keep the pickup cell clear of debris
	if bestY < len(g.debris) && x < len(g.debris[bestY]) {
		g.debris[bestY][x] = false
	}
}

func (g *game) tryCollectHeart() {
	if !g.hasHeart {
		return
	}
	if g.frogX != g.heartX || g.frogY != g.heartY {
		return
	}
	g.hasHeart = false
	g.lives++
	g.heartsCollected++
	g.showHUDNotice("+1 Life")
	g.updateHUD()
	g.lastRenderedScore = -1
}

// placeDiamond puts one ♦ at the center of the top mid safe lane (L10+).
func (g *game) placeDiamond() {
	g.hasDiamond = false
	w, h := g.width, g.height
	if g.level < 10 || w <= 0 || h <= 0 {
		return
	}
	// Top mid safe lane = uppermost mid gap (closest to goal, not the goal shoulder)
	topY := -1
	for y := 0; y < h; y++ {
		if y == g.safeTopY || y == g.safeBottomY {
			continue
		}
		if y >= len(g.safeRow) || !g.safeRow[y] {
			continue
		}
		topY = y
		break
	}
	if topY < 0 {
		return
	}
	x := w / 2
	// Don't stack on the ♥ if both land on the same cell
	if g.hasHeart && g.heartX == x && g.heartY == topY {
		x++
		if x >= w {
			x = w/2 - 1
		}
		if x < 0 {
			x = 0
		}
	}
	g.diamondX, g.diamondY = x, topY
	g.hasDiamond = true
	if topY < len(g.debris) && x < len(g.debris[topY]) {
		g.debris[topY][x] = false
	}
}

func (g *game) tryCollectDiamond() {
	if !g.hasDiamond {
		return
	}
	if g.frogX != g.diamondX || g.frogY != g.diamondY {
		return
	}
	g.hasDiamond = false
	g.score += 1000
	if g.score > g.topScore {
		g.topScore = g.score
	}
	g.gemsCollected++
	g.showHUDNotice("+1,000 Points")
	g.updateHUD()
	g.lastRenderedScore = -1
}

func (g *game) showHUDNotice(msg string) {
	g.hudNoticeText = msg
	g.hudNoticeUntil = time.Now().Add(time.Second)
}


func (g *game) isDebris(x, y int) bool {
	if y < 0 || y >= len(g.debris) {
		return false
	}
	if x < 0 || x >= len(g.debris[y]) {
		return false
	}
	return g.debris[y][x]
}

func (g *game) handleInput(e *tcell.EventKey) bool {
	// Handle start screen
	if g.showStartScreen {
		return g.handleStartInput(e)
	}
	// Return-to-menu confirmation (opened by Esc)
	if g.confirmMenu {
		return g.handleConfirmMenuInput(e)
	}
	// Ignore inputs briefly after death/level clear; drain any backlog (key-repeat
	// under lag) so carried presses don't hop Larry once the gate opens.
	if time.Now().Before(g.acceptInputAfter) {
		g.flushInput()
		return false
	}
	if g.enteringName {
		// Simple name input handler
		switch e.Key() {
		case tcell.KeyEnter:
			g.commitScoreName()
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
			// resuming — drop any key-repeat that piled up while paused
			g.paused = false
			g.flushInput()
			g.screen.Sync()
			if g.scoreTimerActive {
				g.nextScoreDecrement = time.Now().Add(time.Second)
			}
		} else {
			// pausing — draw overlay once; main loop skips further redraws while paused
			g.paused = true
			g.render()
		}
		return false
	}
	if g.paused {
		return false
	}
	prevX, prevY := g.frogX, g.frogY
	prevHighest, prevScore, prevTop := g.highestY, g.score, g.topScore
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
			g.score += (g.highestY - g.frogY) * climbPointsPerRow(g.level)
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
				g.score += (g.highestY - g.frogY) * climbPointsPerRow(g.level)
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
	// Impassable debris on safe gaps — stay put (and undo any score from a blocked step)
	if moved && g.isDebris(g.frogX, g.frogY) {
		g.frogX, g.frogY = prevX, prevY
		g.highestY, g.score, g.topScore = prevHighest, prevScore, prevTop
		moved = false
	}
	if moved {
		g.tryCollectHeart()
		g.tryCollectDiamond()
	}
	if moved && !g.scoreTimerActive {
		g.scoreTimerActive = true
		g.nextScoreDecrement = time.Now().Add(time.Second)
	}
	return false
}

func (g *game) handleStartInput(e *tcell.EventKey) bool {
	if g.startView == startScores {
		n := len(g.highScores)
		switch e.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			g.startView = startMenu
			return false
		case tcell.KeyUp:
			if n > 0 {
				g.scoreIndex = (g.scoreIndex - 1 + n) % n
			}
			return false
		case tcell.KeyDown:
			if n > 0 {
				g.scoreIndex = (g.scoreIndex + 1) % n
			}
			return false
		case tcell.KeyRune:
			switch e.Rune() {
			case ' ':
				g.startView = startMenu
			case 'w', 'W':
				if n > 0 {
					g.scoreIndex = (g.scoreIndex - 1 + n) % n
				}
			case 's', 'S':
				if n > 0 {
					g.scoreIndex = (g.scoreIndex + 1) % n
				}
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
		g.scoreIndex = 0
		return false
	case 2: // Quit
		return true
	}
	return false
}

func (g *game) handleConfirmMenuInput(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEnter:
		if g.confirmYes {
			g.returnToMainMenu()
		} else {
			g.confirmMenu = false
		}
		return false
	case tcell.KeyLeft:
		g.confirmYes = true
		return false
	case tcell.KeyRight:
		g.confirmYes = false
		return false
	case tcell.KeyRune:
		switch e.Rune() {
		case 'a', 'A':
			g.confirmYes = true
		case 'd', 'D':
			g.confirmYes = false
		}
	}
	return false
}

func (g *game) returnToMainMenu() {
	g.confirmMenu = false
	g.paused = false
	g.enteringName = false
	g.nameBuffer = ""
	g.gameOver = false
	g.resetGame()
}

func (g *game) clampFrog() {
	if g.frogX < 0 {
		g.frogX = 0
	}
	if g.frogX >= g.width {
		g.frogX = max(0, g.width-1)
	}
	// Cannot enter the HUD row; top of playfield is the goal shoulder
	if g.frogY < g.safeTopY {
		g.frogY = g.safeTopY
	}
	if g.frogY > g.safeBottomY {
		g.frogY = g.safeBottomY
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
	if g.confirmMenu {
		return
	}
	if g.enteringName {
		return
	}
	// While the post-level / death input gate is up, keep draining the event
	// pipeline every tick so queued moves can't fire after the gate opens.
	if time.Now().Before(g.acceptInputAfter) {
		g.flushInput()
	}
	g.advanceLanes()

	// Collision detection with lanes (ignore safe rows)
	isSafe := g.frogY >= 0 && g.frogY < len(g.safeRow) && g.safeRow[g.frogY]
	if !isSafe {
		for _, ln := range g.lanes {
			if ln.y == g.frogY {
				for _, cx := range ln.cars {
					if carCoversCell(cx, ln.length, ln.width, g.frogX) {
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

	// Reached goal at (or above) top safe shoulder — advance level
	if g.frogY <= g.safeTopY {
		g.score += 100 * g.level
		if g.score > g.topScore {
			g.topScore = g.score
		}
		g.nextLevel()
		return
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
		if y == g.safeTopY {
			// Checker pattern marks the goal line clearly
			st := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(g.theme.goal).Bold(true)
			for x := 0; x < w; x++ {
				ch := glyphGoalA
				if x%2 == 1 {
					ch = glyphGoalB
				}
				g.screen.SetContent(x, y, ch, nil, st)
			}
			continue
		}
		var bg tcell.Color
		if y == g.safeBottomY || (y >= 0 && y < len(g.safeRow) && g.safeRow[y]) {
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

// carCoversCell reports whether a wrapping vehicle occupies column x.
func carCoversCell(left, length, width, x int) bool {
	if width <= 0 || length <= 0 {
		return false
	}
	for dx := 0; dx < length; dx++ {
		cx := (left + dx) % width
		if cx < 0 {
			cx += width
		}
		if cx == x {
			return true
		}
	}
	return false
}

func (g *game) drawVehicles() {
	w, h := g.width, g.height
	for _, ln := range g.lanes {
		st := tcell.StyleDefault.Foreground(ln.color)
		lw := ln.width
		if lw <= 0 {
			lw = w
		}
		for _, left := range ln.cars {
			for dx := 0; dx < ln.length; dx++ {
				x := (left + dx) % lw
				if x < 0 {
					x += lw
				}
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

func (g *game) drawDebris() {
	st := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(g.theme.safe).Bold(true)
	for y := 0; y < len(g.debris); y++ {
		for x := 0; x < len(g.debris[y]); x++ {
			if g.debris[y][x] {
				g.screen.SetContent(x, y, glyphDebris, nil, st)
			}
		}
	}
}

func (g *game) drawHeart() {
	if !g.hasHeart {
		return
	}
	fg := tcell.ColorRed
	// Flash white/red ~4 Hz for eye catch
	if time.Now().UnixNano()/int64(250*time.Millisecond)%2 == 0 {
		fg = tcell.ColorWhite
	}
	st := tcell.StyleDefault.Foreground(fg).Background(g.theme.safe).Bold(true)
	g.screen.SetContent(g.heartX, g.heartY, glyphHeart, nil, st)
}

func (g *game) drawDiamond() {
	if !g.hasDiamond {
		return
	}
	fg := tcell.ColorAqua
	// Flash white/cyan ~4 Hz for eye catch
	if time.Now().UnixNano()/int64(250*time.Millisecond)%2 == 0 {
		fg = tcell.ColorWhite
	}
	st := tcell.StyleDefault.Foreground(fg).Background(g.theme.safe).Bold(true)
	g.screen.SetContent(g.diamondX, g.diamondY, glyphDiamond, nil, st)
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
	g.drawDebris()
	g.drawHeart()
	g.drawDiamond()
	g.drawVehicles()

	// Draw HUD - will refresh only when score changes
	if g.score != g.lastRenderedScore {
		g.updateHUD()
		g.lastRenderedScore = g.score
	}
	if time.Now().Before(g.hudNoticeUntil) {
		// Invert HUD colors and center pickup notice (life / points)
		invStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(tcell.ColorBlack).Bold(true)
		drawText(s, 0, 0, spaces(w), invStyle)
		msg := g.hudNoticeText
		if msg == "" {
			msg = "+1 Life"
		}
		drawCentered(s, w/2, 0, msg, invStyle)
	} else {
		// HUD uses Larry's highlight so it stays distinct from the playfield
		hudStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.theme.frog).Bold(true)
		drawText(s, 0, 0, spaces(w), hudStyle)
		drawText(s, 0, 0, g.hudLine, hudStyle)
	}

	// Larry uses theme.frog (always yellow or lime highlight — never a vehicle hue)
	frogStyle := tcell.StyleDefault.Foreground(g.theme.frog).Bold(true)
	s.SetContent(g.frogX, g.frogY, glyphLarry, nil, frogStyle)

	// Ensure overlays are drawn last, on top of vehicles and frog
	if g.confirmMenu {
		g.drawConfirmMenuOverlay()
	} else if g.enteringName {
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
	if !g.testMode {
		now := time.Now()
		entry := scoreEntry{
			Name:    name,
			Score:   g.score,
			Time:    now.Unix(),
			Date:    now.Format("010206"),
			Level:   g.level,
			Hearts:  g.heartsCollected,
			Gems:    g.gemsCollected,
			Version: larryVersion,
		}
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
	}
	g.enteringName = false
	g.resetGame()
}

func (g *game) resetGame() {
	g.lives = 3
	g.score = 0
	g.heartsCollected = 0
	g.gemsCollected = 0
	g.lastRenderedScore = -1
	g.level = 1
	g.theme = themeForLevel(g.level)
	g.createLanes()
	g.frogX = g.width / 2
	g.frogY = g.safeBottomY
	g.highestY = g.frogY
	g.gameOver = false
	g.confirmMenu = false
	g.paused = false
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
	if json.Unmarshal(data, &list) != nil {
		return
	}
	g.highScores = list
	if migrateScoreEntries(g.highScores) {
		g.saveHighScores()
	}
}

// migrateScoreEntries fills missing v1.5+ fields on legacy entries.
// Returns true if the file should be rewritten.
func migrateScoreEntries(list []scoreEntry) bool {
	changed := false
	for i := range list {
		if list[i].Version != "" {
			continue
		}
		list[i].Version = "pre-1.5"
		changed = true
	}
	return changed
}

func (g *game) saveHighScores() {
	if g.testMode {
		return
	}
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
			// drop queued keys (and other events) so lag/key-repeat can't carry over
		default:
			return
		}
	}
}

func (g *game) handleQuit(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyCtrlC {
		return true
	}
	if e.Key() != tcell.KeyEscape {
		return false
	}
	// Esc on high scores returns to the start menu
	if g.showStartScreen && g.startView == startScores {
		return false
	}
	// Esc on the main menu exits the app
	if g.showStartScreen {
		return true
	}
	// Esc during confirm cancels and resumes
	if g.confirmMenu {
		g.confirmMenu = false
		g.flushInput()
		return false
	}
	// Esc during play / name entry → confirm return to menu (Larry starts on No)
	g.confirmMenu = true
	g.confirmYes = false
	g.render() // draw once; main loop skips redraws while confirm is open
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
	help := "  (Space:Pause Esc:Menu)"
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

func (g *game) drawConfirmMenuOverlay() {
	w, h := g.width, g.height
	if w <= 0 || h <= 0 {
		return
	}
	boxBg := tcell.ColorBlack
	borderStyle := tcell.StyleDefault.Foreground(g.theme.frog).Background(boxBg).Bold(true)
	fillStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg).Bold(true)
	hintStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(boxBg)
	// Larry uses hint aqua so he stands out from Yes/No (white).
	larryStyle := hintStyle.Bold(true)
	inner := 28
	topY := h/2 - 4
	if topY < 0 {
		topY = 0
	}
	// Larry hops under Yes (left) or No (right); defaults to No
	var choiceLine string
	if g.confirmYes {
		choiceLine = "  " + string(glyphLarry) + " Yes            No  "
	} else {
		choiceLine = "    Yes          " + string(glyphLarry) + " No  "
	}
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, fillStyle,
		[]string{
			"Return to main menu?",
			"",
			choiceLine,
			"",
			"←→ hop   Enter select",
			"Esc cancels",
		},
		[]tcell.Style{fillStyle, fillStyle, fillStyle, fillStyle, hintStyle, hintStyle})
	// Repaint Larry in aqua; the choice row was drawn white for Yes/No contrast.
	choiceRunes := []rune(choiceLine)
	larryIdx := -1
	for i, r := range choiceRunes {
		if r == glyphLarry {
			larryIdx = i
			break
		}
	}
	if larryIdx >= 0 {
		boxW := inner + 2
		left := w/2 - boxW/2
		if left < 0 {
			left = 0
		}
		pad := (inner - len(choiceRunes)) / 2
		if pad < 0 {
			pad = 0
		}
		choiceY := topY + 1 + 2
		g.screen.SetContent(left+1+pad+larryIdx, choiceY, glyphLarry, nil, larryStyle)
	}
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
	list = append(list, scoreEntry{
		Name:    "YOUR SCORE",
		Score:   g.score,
		Time:    now.Unix(),
		Date:    now.Format("010206"),
		Level:   g.level,
		Hearts:  g.heartsCollected,
		Gems:    g.gemsCollected,
		Version: larryVersion,
	})
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
	// theme.frog is UI chrome (HUD/borders); Larry on the field always uses larryHighlight
	// so he never shares a vehicle color. Avoid green frog on green-car themes.
	palettes := []theme{
		{bg: tcell.ColorReset, fg: tcell.ColorWhite, road: tcell.ColorGray, river: tcell.ColorNavy, safe: tcell.ColorDarkOliveGreen, frog: larryHighlight, carSmall: tcell.ColorLightSalmon, carRegular: tcell.ColorOrangeRed, carSemi: tcell.ColorTomato, log: tcell.ColorSandyBrown, goal: tcell.ColorDarkCyan},
		{bg: tcell.ColorBlack, fg: tcell.ColorLightCyan, road: tcell.ColorDarkSlateGray, river: tcell.ColorBlue, safe: tcell.ColorDarkGreen, frog: larryHighlight, carSmall: tcell.ColorLightSkyBlue, carRegular: tcell.ColorSteelBlue, carSemi: tcell.ColorRoyalBlue, log: tcell.ColorBurlyWood, goal: tcell.ColorDarkTurquoise},
		{bg: tcell.ColorBlack, fg: tcell.ColorWhite, road: tcell.ColorDimGray, river: tcell.ColorDarkBlue, safe: tcell.ColorDarkOliveGreen, frog: larryHighlight, carSmall: tcell.ColorPlum, carRegular: tcell.ColorMediumVioletRed, carSemi: tcell.ColorDeepPink, log: tcell.ColorPeru, goal: tcell.ColorTeal},
		{bg: tcell.ColorBlack, fg: tcell.ColorSilver, road: tcell.ColorGray, river: tcell.ColorDarkSlateBlue, safe: tcell.ColorDarkGreen, frog: tcell.ColorLime, carSmall: tcell.ColorKhaki, carRegular: tcell.ColorGoldenrod, carSemi: tcell.ColorSaddleBrown, log: tcell.ColorTan, goal: tcell.ColorCadetBlue},
		{bg: tcell.ColorBlack, fg: tcell.ColorWhite, road: tcell.ColorGray, river: tcell.ColorRoyalBlue, safe: tcell.ColorDarkOliveGreen, frog: larryHighlight, carSmall: tcell.ColorLightGreen, carRegular: tcell.ColorSeaGreen, carSemi: tcell.ColorDarkGreen, log: tcell.ColorSandyBrown, goal: tcell.ColorSteelBlue},
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
	g.drawDebris()
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
	maxLabel := 0
	for _, label := range menuItems {
		if n := len([]rune(label)); n > maxLabel {
			maxLabel = n
		}
	}

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
		// Same-length rows so centered lines keep first letters (and Larry) aligned
		padLabel := label + strings.Repeat(" ", maxLabel-len([]rune(label)))
		cursor := "  "
		st := normalStyle
		if i == g.menuIndex {
			st = selectedStyle
			if blinkOn {
				cursor = string(glyphLarry) + " "
			}
		}
		lines = append(lines, cursor+padLabel)
		styles = append(styles, st)
	}
	lines = append(lines, boxDivider)
	styles = append(styles, fillStyle)
	lines = append(lines, "↑↓ hop Larry  ·  Enter Confirm")
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
	detailStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(boxBg)
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
		if g.scoreIndex < 0 {
			g.scoreIndex = 0
		}
		if g.scoreIndex >= show {
			g.scoreIndex = show - 1
		}
		for i := 0; i < show; i++ {
			e := g.highScores[i]
			cursor := "  "
			if i == g.scoreIndex {
				cursor = string(glyphLarry) + " "
			}
			lines = append(lines, fmt.Sprintf("%s%2d. %-8s  %6d  %s", cursor, i+1, e.Name, e.Score, e.Date))
			if i == g.scoreIndex {
				styles = append(styles, champStyle)
			} else {
				styles = append(styles, fillStyle)
			}
		}
		// Details for the selected run — omit blank / unknown fields
		sel := g.highScores[g.scoreIndex]
		details := scoreDetailLines(sel)
		if len(details) > 0 {
			lines = append(lines, boxDivider)
			styles = append(styles, fillStyle)
			for _, d := range details {
				lines = append(lines, d)
				styles = append(styles, detailStyle)
			}
		}
	}
	lines = append(lines, boxDivider)
	styles = append(styles, fillStyle)
	if len(g.highScores) > 0 {
		lines = append(lines, "↑↓ hop Larry   Esc/Enter return")
	} else {
		lines = append(lines, "Esc or Enter to return")
	}
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
	drawBorderedBox(g.screen, w/2, topY, inner, borderStyle, fillStyle, lines, styles)
}

// scoreDetailLines returns non-blank saved metadata for a high-score entry.
func scoreDetailLines(e scoreEntry) []string {
	var d []string
	if e.Version != "" && e.Version != "pre-1.5" {
		d = append(d, "Version: "+e.Version)
	}
	if e.Level > 0 {
		d = append(d, fmt.Sprintf("Level died: %d", e.Level))
	}
	if e.Hearts > 0 {
		d = append(d, "Hearts: "+strings.Repeat(string(glyphHeart), e.Hearts))
	}
	if e.Gems > 0 {
		d = append(d, "Gems: "+strings.Repeat(string(glyphDiamond), e.Gems))
	}
	return d
}

