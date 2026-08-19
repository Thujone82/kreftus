package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

const saveDebounce = 300 * time.Millisecond

const (
	viewMenu = iota
	viewPlay
	viewPaused
	viewConfirmNewGame
	viewSolved
)

const (
	menuContinue = iota
	menuNewGame
	menuQuit
)

type game struct {
	screen tcell.Screen
	width  int
	height int

	view         int
	menuIndex    int
	confirmIndex int // new game: 0 Cancel / 1 Abandon

	difficulty string
	save       *saveData
	rng        *rand.Rand

	puzzle         puzzleEntry
	board          board
	solvedMs       int64
	solvedMistakes int

	elapsed      time.Duration
	clockAnchor  time.Time
	clockRunning bool

	accentIndex   int
	flashUntil    time.Time
	flashOK       bool
	pendingSolved bool
	redraw        chan struct{}

	pencil        bool      // Tab: false = pen ✒️, true = pencil ✏️
	shiftHold     bool      // momentary pencil while Shift is down; not saved
	lastShiftHeld time.Time // last time Shift was actually down (NumLock fakes a Shift-up)

	events    chan tcell.Event
	saveFlush chan struct{}
	saveTimer *time.Timer
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Print("\033[0m")
			fmt.Print("\033[?25h")
			panic(r)
		}
	}()

	enableUTF8Console()
	resizeToPreferred()
	startShiftWatch()
	defer stopShiftWatch()

	if err := loadPuzzleBank(); err != nil {
		fmt.Fprintf(os.Stderr, "sudoku: load puzzles: %v\n", err)
		os.Exit(1)
	}

	s, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := s.Init(); err != nil {
		panic(err)
	}
	enableUTF8Console()
	resizeToPreferred()
	s.Sync()
	s.DisableMouse()
	s.DisablePaste()
	s.Clear()
	s.HideCursor()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cleanup := func() {
		fmt.Print("\033[0m")
		fmt.Print("\033[?25h")
		s.Fini()
		fmt.Print("\033[H\033[2J") // Clear-Host after leaving the TUI
	}
	defer cleanup()

	setTerminalTitle("Sudoku")

	g := &game{
		screen:     s,
		view:       viewMenu,
		menuIndex:  menuNewGame,
		difficulty: diffEasy,
		save:       loadSave(),
		rng:        rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(os.Getpid()))),
		redraw:     make(chan struct{}, 1),
		saveFlush:  make(chan struct{}, 1),
	}
	g.accentIndex = g.rng.IntN(colorWheelSize)
	g.width, g.height = s.Size()
	if d := g.save.firstContinueDifficulty(); d != "" {
		g.difficulty = d
		g.menuIndex = menuContinue
	}

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			switch ev.(type) {
			case *tcell.EventKey, *tcell.EventResize:
				select {
				case events <- ev:
				default:
				}
			}
		}
	}()
	g.events = events

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	shiftTick := time.NewTicker(50 * time.Millisecond)
	defer shiftTick.Stop()

	g.render()

	handleEv := func(ev tcell.Event) bool {
		switch e := ev.(type) {
		case *tcell.EventResize:
			g.resize(e)
			g.render()
		case *tcell.EventKey:
			if g.handleQuit(e) {
				g.persistIfPlaying()
				return true
			}
			if g.handleInput(e) {
				g.persistIfPlaying()
				return true
			}
			g.render()
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
			if g.flashing() {
				g.maybeShowSolved()
				g.render()
			} else if g.view == viewPlay && g.clockRunning {
				g.drawHUD()
				g.screen.Show()
			}
		case <-shiftTick.C:
			if g.view == viewPlay && !g.pendingSolved && g.refreshShiftHold() {
				g.render()
			}
		case <-g.redraw:
			g.maybeShowSolved()
			g.render()
		case <-g.saveFlush:
			g.flushSave()
		case <-sigChan:
			g.persistIfPlaying()
			return
		}
	}
}

func (g *game) resize(e *tcell.EventResize) {
	g.screen.Sync()
	g.width, g.height = e.Size()
}

func (g *game) handleQuit(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyCtrlC {
		return true
	}
	return false
}

func (g *game) handleInput(e *tcell.EventKey) bool {
	switch g.view {
	case viewMenu:
		return g.handleMenu(e)
	case viewPlay:
		return g.handlePlay(e)
	case viewPaused:
		return g.handlePaused(e)
	case viewConfirmNewGame:
		return g.handleConfirm(e)
	case viewSolved:
		return g.handleSolved(e)
	}
	return false
}

func (g *game) handleMenu(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEscape:
		return true
	case tcell.KeyUp:
		g.menuMove(-1)
		g.rotateAccent(1)
	case tcell.KeyDown:
		g.menuMove(1)
		g.rotateAccent(1)
	case tcell.KeyLeft:
		g.difficulty = nextDifficulty(g.difficulty, -1)
		g.clampMenu()
		g.rotateAccent(1)
	case tcell.KeyRight:
		g.difficulty = nextDifficulty(g.difficulty, 1)
		g.clampMenu()
		g.rotateAccent(1)
	case tcell.KeyEnter:
		return g.activateMenu()
	case tcell.KeyRune:
		switch e.Rune() {
		case 'w', 'W', 'k', 'K':
			g.menuMove(-1)
			g.rotateAccent(1)
		case 's', 'S', 'j', 'J':
			g.menuMove(1)
			g.rotateAccent(1)
		case 'a', 'A', 'h', 'H':
			g.difficulty = nextDifficulty(g.difficulty, -1)
			g.clampMenu()
			g.rotateAccent(1)
		case 'd', 'D', 'l', 'L':
			g.difficulty = nextDifficulty(g.difficulty, 1)
			g.clampMenu()
			g.rotateAccent(1)
		}
	}
	return false
}

func (g *game) visibleMenu() []int {
	items := make([]int, 0, 4)
	if g.save.continueFor(g.difficulty) != nil {
		items = append(items, menuContinue)
	}
	return append(items, menuNewGame, menuQuit)
}

func (g *game) menuMove(dir int) {
	items := g.visibleMenu()
	idx := 0
	for i, id := range items {
		if id == g.menuIndex {
			idx = i
			break
		}
	}
	n := len(items)
	idx = (idx + dir + n) % n
	g.menuIndex = items[idx]
}

func (g *game) clampMenu() {
	for _, id := range g.visibleMenu() {
		if id == g.menuIndex {
			return
		}
	}
	g.menuIndex = menuNewGame
}

func (g *game) activateMenu() bool {
	switch g.menuIndex {
	case menuContinue:
		if g.save.continueFor(g.difficulty) == nil {
			return false
		}
		g.resumeContinue()
	case menuNewGame:
		if remainingCount(g.difficulty, g.save.completedSet(g.difficulty)) == 0 {
			return false
		}
		if g.save.continueFor(g.difficulty) != nil {
			g.view = viewConfirmNewGame
			g.confirmIndex = 0
			return false
		}
		g.startNewGame()
	case menuQuit:
		return true
	}
	return false
}

func (g *game) handlePlay(e *tcell.EventKey) bool {
	if g.pendingSolved || g.board.isComplete() {
		return false
	}
	g.noteShift(e)
	if g.shiftHold {
		if d, ok := numpadShiftDigit(e.Key()); ok && shiftMarksKey(e.Key(), keypadOrigin()) {
			g.applyDigit(d)
			return false
		}
		if e.Key() == tcell.KeyInsert && keypadOrigin() {
			if g.clearPlay() {
				g.persistPlay()
			}
			return false
		}
	}
	switch e.Key() {
	case tcell.KeyEscape:
		g.returnToMenu()
		return false
	case tcell.KeyTab, tcell.KeyBacktab:
		g.pencil = !g.pencil
		g.persistPlay()
	case tcell.KeyLeft:
		g.movePlay(-1, 0)
	case tcell.KeyRight:
		g.movePlay(1, 0)
	case tcell.KeyUp:
		g.movePlay(0, -1)
	case tcell.KeyDown:
		g.movePlay(0, 1)
	case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
		if g.clearPlay() {
			g.persistPlay()
		}
	case tcell.KeyRune:
		r := e.Rune()
		if d, ok := digitFromRune(r); ok {
			g.applyDigit(d)
			break
		}
		switch r {
		case ' ':
			g.stopClock()
			g.persistPlayNow()
			g.view = viewPaused
			return false
		case '\t':
			g.pencil = !g.pencil
			g.persistPlay()
		case 'a', 'A', 'h', 'H':
			g.movePlay(-1, 0)
		case 'd', 'D', 'l', 'L':
			g.movePlay(1, 0)
		case 'w', 'W', 'k', 'K':
			g.movePlay(0, -1)
		case 's', 'S', 'j', 'J':
			g.movePlay(0, 1)
		case '0', '.':
			if g.clearPlay() {
				g.persistPlay()
			}
		}
	}
	return false
}

func (g *game) applyDigit(d byte) {
	if g.pencilActive() {
		if g.board.markPencil(d) {
			g.rotateAccent(g.accentStep())
			g.persistPlay()
		}
		return
	}
	if g.board.place(d) {
		if g.board.isLocked(g.board.cursor) {
			g.rotateAccent(1)
			g.startFlash(true)
		} else {
			g.rotateAccent(-8)
			g.startFlash(false)
		}
		if g.board.isComplete() {
			g.finishSuccess()
		} else {
			g.persistPlay()
		}
	}
}

func (g *game) pencilActive() bool {
	return g.pencil || g.shiftHold
}

const shiftReleaseGrace = 150 * time.Millisecond

func (g *game) markShiftHeld() {
	g.shiftHold = true
	g.lastShiftHeld = time.Now()
}

func (g *game) shiftRecentlyHeld() bool {
	return !g.lastShiftHeld.IsZero() && time.Since(g.lastShiftHeld) < shiftReleaseGrace
}

func (g *game) noteShift(e *tcell.EventKey) {
	if shiftHeld() || e.Modifiers()&tcell.ModShift != 0 || (e.Key() == tcell.KeyRune && isShiftedDigit(e.Rune())) {
		g.markShiftHeld()
		return
	}
	// NumLock+Shift synthesizes a Shift-up around the keypad event, so
	// neither GetAsyncKeyState nor ModShift is set. Keep pencil for nav keys.
	if _, isPad := numpadShiftDigit(e.Key()); isPad || e.Key() == tcell.KeyInsert {
		if g.shiftHold || g.shiftRecentlyHeld() {
			g.shiftHold = true
			return
		}
	}
	g.shiftHold = false
}

func (g *game) refreshShiftHold() bool {
	if !shiftPollable() {
		return false
	}
	if shiftHeld() {
		changed := !g.shiftHold
		g.markShiftHeld()
		return changed
	}
	if g.shiftHold && g.shiftRecentlyHeld() {
		return false
	}
	if g.shiftHold {
		g.shiftHold = false
		return true
	}
	return false
}

func digitFromRune(r rune) (byte, bool) {
	if r >= '1' && r <= '9' {
		return byte(r), true
	}
	if d, ok := shiftedDigit[r]; ok {
		return d, true
	}
	return 0, false
}

func isShiftedDigit(r rune) bool {
	_, ok := shiftedDigit[r]
	return ok
}

// Windows NumLock+Shift turns the keypad into nav keys (End, arrows, Clear, …).
// Map those back to 1–9 so Shift+numpad marks. Dedicated arrows still move
// (they are extended keys); WASD also moves while Shift is down.
func numpadShiftDigit(k tcell.Key) (byte, bool) {
	switch k {
	case tcell.KeyEnd:
		return '1', true
	case tcell.KeyDown:
		return '2', true
	case tcell.KeyPgDn:
		return '3', true
	case tcell.KeyLeft:
		return '4', true
	case tcell.KeyClear, tcell.KeyCenter:
		return '5', true
	case tcell.KeyRight:
		return '6', true
	case tcell.KeyHome:
		return '7', true
	case tcell.KeyUp:
		return '8', true
	case tcell.KeyPgUp:
		return '9', true
	}
	return 0, false
}

func isArrowKey(k tcell.Key) bool {
	return k == tcell.KeyUp || k == tcell.KeyDown || k == tcell.KeyLeft || k == tcell.KeyRight
}

// shiftMarksKey is true for Shift+numpad digits. Dedicated arrows are extended
// keys (keypadOrigin false) and still move the cursor.
func shiftMarksKey(k tcell.Key, fromKeypad bool) bool {
	if _, ok := numpadShiftDigit(k); !ok {
		return false
	}
	if isArrowKey(k) && !fromKeypad {
		return false
	}
	return true
}

// US-layout Shift+1..9. Terminals often deliver these instead of '1'-'9' with ModShift.
var shiftedDigit = map[rune]byte{
	'!': '1', '@': '2', '#': '3', '$': '4', '%': '5',
	'^': '6', '&': '7', '*': '8', '(': '9',
}

func (g *game) movePlay(dx, dy int) {
	g.board.move(dx, dy)
	g.rotateAccent(g.accentStep())
}

func (g *game) accentStep() int {
	if g.pencilActive() {
		return -1
	}
	return 1
}

func (g *game) clearPlay() bool {
	if g.pencilActive() {
		return g.board.clearPencil()
	}
	return g.board.clear()
}

func (g *game) handlePaused(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEscape {
		g.returnToMenu()
		return false
	}
	if e.Key() == tcell.KeyRune && e.Rune() == ' ' {
		g.view = viewPlay
		g.startClock()
		g.persistPlayNow()
	}
	return false
}

func (g *game) handleConfirm(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEscape:
		g.cancelConfirm()
	case tcell.KeyLeft:
		g.confirmIndex = 0
	case tcell.KeyRight:
		g.confirmIndex = 1
	case tcell.KeyEnter:
		return g.acceptConfirm()
	case tcell.KeyRune:
		switch e.Rune() {
		case 'h', 'H':
			g.confirmIndex = 0
		case 'l', 'L':
			g.confirmIndex = 1
		case 'a', 'A':
			if g.view == viewConfirmNewGame {
				g.confirmIndex = 1
				return g.acceptConfirm()
			}
		case 'c', 'C':
			if g.view == viewConfirmNewGame {
				g.cancelConfirm()
			}
		}
	}
	return false
}

func (g *game) cancelConfirm() {
	g.view = viewMenu
}

func (g *game) acceptConfirm() bool {
	if g.view != viewConfirmNewGame {
		return false
	}
	if g.confirmIndex == 0 {
		g.cancelConfirm()
		return false
	}
	g.abandonContinue()
	g.startNewGame()
	return false
}

func (g *game) returnToMenu() {
	g.stopClock()
	if g.shouldPersistContinue() {
		g.persistPlayNow()
	}
	g.puzzle = puzzleEntry{}
	g.view = viewMenu
	if g.save.continueFor(g.difficulty) != nil {
		g.menuIndex = menuContinue
	} else {
		g.menuIndex = menuNewGame
	}
}

func (g *game) handleSolved(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEnter || e.Key() == tcell.KeyEscape || (e.Key() == tcell.KeyRune && e.Rune() == ' ') {
		g.view = viewMenu
		g.menuIndex = menuNewGame
	}
	return false
}

func (g *game) startClock() {
	if !g.clockRunning {
		g.clockAnchor = time.Now()
		g.clockRunning = true
	}
}

func (g *game) stopClock() {
	if g.clockRunning {
		g.elapsed += time.Since(g.clockAnchor)
		g.clockRunning = false
	}
}

func (g *game) currentElapsed() time.Duration {
	e := g.elapsed
	if g.clockRunning {
		e += time.Since(g.clockAnchor)
	}
	return e
}

func (g *game) persistIfPlaying() {
	if g.view == viewPlay || g.view == viewPaused {
		g.stopClock()
		if g.shouldPersistContinue() {
			g.persistPlayNow()
			return
		}
	}
	if g.saveTimer != nil {
		g.flushSave()
	}
}

func (g *game) shouldPersistContinue() bool {
	if g.puzzle.ID == "" || g.pendingSolved || g.view == viewSolved {
		return false
	}
	return !g.board.isComplete()
}

func (g *game) persistPlay() {
	if !g.shouldPersistContinue() {
		return
	}
	g.storeContinue()
	g.scheduleSave()
}

func (g *game) persistPlayNow() {
	if !g.shouldPersistContinue() {
		return
	}
	g.storeContinue()
	g.flushSave()
}

func (g *game) storeContinue() {
	ms := g.currentElapsed().Milliseconds()
	top, bot, slot := g.board.pencilsString()
	g.save.setContinue(g.difficulty, &continueGame{
		ID:         g.puzzle.ID,
		Difficulty: g.difficulty,
		Givens:     g.puzzle.Givens,
		Solution:   g.puzzle.Solution,
		Grid:       g.board.gridString(),
		ElapsedMs:  ms,
		Mistakes:   g.board.mistakes,
		Pencil:     g.pencil,
		PencilTop:  top,
		PencilBot:  bot,
		PencilSlot: slot,
	})
}

func (g *game) scheduleSave() {
	if g.saveFlush == nil {
		_ = g.save.write()
		return
	}
	if g.saveTimer != nil {
		g.saveTimer.Stop()
	}
	g.saveTimer = time.AfterFunc(saveDebounce, func() {
		select {
		case g.saveFlush <- struct{}{}:
		default:
		}
	})
}

func (g *game) flushSave() {
	if g.saveTimer != nil {
		g.saveTimer.Stop()
		g.saveTimer = nil
	}
	if g.save == nil {
		return
	}
	_ = g.save.write()
}

func (g *game) startNewGame() {
	p, ok := pickIncomplete(g.difficulty, g.save.completedSet(g.difficulty), g.rng)
	if !ok {
		g.view = viewMenu
		return
	}
	if !p.ensureSolved() {
		g.view = viewMenu
		return
	}
	g.rotateAccent(1)
	g.puzzle = p
	g.board = newBoard(p.Givens, p.Solution, p.Givens)
	g.pencil = false
	g.elapsed = 0
	g.clockRunning = false
	g.startClock()
	g.view = viewPlay
	g.persistPlayNow()
}

func (g *game) resumeContinue() {
	c := g.save.continueFor(g.difficulty)
	if c == nil {
		return
	}
	g.difficulty = c.Difficulty
	p, ok := puzzleByID(c.Difficulty, c.ID)
	if !ok {
		p = puzzleEntry{ID: c.ID, Givens: c.Givens}
	}
	if len(c.Solution) == 81 {
		p.Solution = c.Solution
	} else if !p.ensureSolved() {
		return
	}
	g.puzzle = p
	g.board = newBoard(c.Givens, p.Solution, c.Grid)
	g.board.mistakes = c.Mistakes
	g.board.loadPencils(c.PencilTop, c.PencilBot, c.PencilSlot)
	g.board.stripImpossiblePencils()
	g.pencil = c.Pencil
	g.elapsed = time.Duration(c.ElapsedMs) * time.Millisecond
	g.clockRunning = false
	g.startClock()
	g.view = viewPlay
	if g.board.isComplete() {
		g.finishSuccess()
	}
}

func (g *game) abandonContinue() {
	c := g.save.continueFor(g.difficulty)
	if c == nil {
		return
	}
	g.save.recordFailure(c.Difficulty)
	g.save.setContinue(g.difficulty, nil)
	g.flushSave()
}

func (g *game) finishSuccess() {
	g.stopClock()
	ms := g.currentElapsed().Milliseconds()
	g.solvedMs = ms
	g.solvedMistakes = g.board.mistakes
	g.save.recordSuccess(g.difficulty, ms, g.board.mistakes)
	g.save.markCompleted(g.difficulty, g.puzzle.ID, g.board.mistakes, ms)
	g.save.setContinue(g.difficulty, nil)
	g.flushSave()
	g.puzzle = puzzleEntry{}
	g.pendingSolved = true
	g.maybeShowSolved()
}

func (g *game) maybeShowSolved() {
	if !g.pendingSolved || g.flashing() {
		return
	}
	g.pendingSolved = false
	g.view = viewSolved
	g.puzzle = puzzleEntry{}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
