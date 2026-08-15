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

const (
	viewMenu = iota
	viewPlay
	viewPaused
	viewConfirmExit
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
	confirmIndex int // exit: 0 Abandon / 1 Quit; new game: 0 Cancel / 1 Abandon

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

	pencil bool // Tab: false = pen ✒️, true = pencil ✏️

	events chan tcell.Event
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
	}
	g.accentIndex = g.rng.IntN(colorWheelSize)
	g.width, g.height = s.Size()
	if g.save.Continue != nil {
		g.menuIndex = menuContinue
		g.difficulty = g.save.Continue.Difficulty
		if g.difficulty == "" {
			g.difficulty = diffEasy
		}
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
			if (g.view == viewPlay && g.clockRunning) || g.flashing() {
				g.maybeShowSolved()
				g.render()
			}
		case <-g.redraw:
			g.maybeShowSolved()
			g.render()
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
	case viewConfirmExit, viewConfirmNewGame:
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
		g.rotateAccent(1)
	case tcell.KeyRight:
		g.difficulty = nextDifficulty(g.difficulty, 1)
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
			g.rotateAccent(1)
		case 'd', 'D', 'l', 'L':
			g.difficulty = nextDifficulty(g.difficulty, 1)
			g.rotateAccent(1)
		}
	}
	return false
}

func (g *game) visibleMenu() []int {
	items := make([]int, 0, 4)
	if g.save.Continue != nil {
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
		if g.save.Continue == nil {
			return false
		}
		g.resumeContinue()
	case menuNewGame:
		if len(incompletePool(g.difficulty, g.save.completedSet(g.difficulty))) == 0 {
			return false
		}
		if g.save.Continue != nil {
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
	switch e.Key() {
	case tcell.KeyEscape:
		g.stopClock()
		g.persistPlay()
		g.view = viewConfirmExit
		g.confirmIndex = 1
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
		switch r {
		case ' ':
			g.stopClock()
			g.persistPlay()
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
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if g.pencil {
				if g.board.markPencil(byte(r)) {
					g.rotateAccent(g.accentStep())
					g.persistPlay()
				}
				break
			}
			if g.board.place(byte(r)) {
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
	}
	return false
}

func (g *game) movePlay(dx, dy int) {
	g.board.move(dx, dy)
	g.rotateAccent(g.accentStep())
}

func (g *game) accentStep() int {
	if g.pencil {
		return -1
	}
	return 1
}

func (g *game) clearPlay() bool {
	if g.pencil {
		return g.board.clearPencil()
	}
	return g.board.clear()
}

func (g *game) handlePaused(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEscape {
		g.persistPlay()
		g.view = viewConfirmExit
		g.confirmIndex = 1
		return false
	}
	if e.Key() == tcell.KeyRune && e.Rune() == ' ' {
		g.view = viewPlay
		g.startClock()
		g.persistPlay()
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
			if g.view == viewConfirmExit {
				g.confirmIndex = 0
				return g.acceptConfirm()
			}
			if g.view == viewConfirmNewGame {
				g.confirmIndex = 1
				return g.acceptConfirm()
			}
		case 'q', 'Q':
			if g.view == viewConfirmExit {
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
	if g.view == viewConfirmNewGame {
		g.view = viewMenu
		return
	}
	g.view = viewPlay
	g.startClock()
}

func (g *game) acceptConfirm() bool {
	if g.view == viewConfirmNewGame {
		if g.confirmIndex == 0 {
			g.cancelConfirm()
			return false
		}
		g.abandonContinue()
		g.startNewGame()
		return false
	}
	// viewConfirmExit: 0 Abandon, 1 Quit
	if g.confirmIndex == 1 {
		g.stopClock()
		g.persistPlay()
		return true
	}
	g.abandonInPlay()
	g.view = viewMenu
	g.menuIndex = menuNewGame
	return false
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
	if !g.shouldPersistContinue() {
		return
	}
	if g.view == viewPlay || g.view == viewPaused || g.view == viewConfirmExit {
		g.stopClock()
		g.persistPlay()
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
	ms := g.currentElapsed().Milliseconds()
	top, bot, slot := g.board.pencilsString()
	g.save.Continue = &continueGame{
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
	}
	_ = g.save.write()
}

func (g *game) startNewGame() {
	pool := incompletePool(g.difficulty, g.save.completedSet(g.difficulty))
	p, ok := pickRandom(pool, g.rng)
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
	g.persistPlay()
}

func (g *game) resumeContinue() {
	c := g.save.Continue
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
	g.pencil = false
	g.elapsed = time.Duration(c.ElapsedMs) * time.Millisecond
	g.clockRunning = false
	g.startClock()
	g.view = viewPlay
	if g.board.isComplete() {
		g.finishSuccess()
	}
}

func (g *game) abandonContinue() {
	c := g.save.Continue
	if c == nil {
		return
	}
	g.save.recordFailure(c.Difficulty)
	g.save.Continue = nil
	_ = g.save.write()
}

func (g *game) abandonInPlay() {
	g.stopClock()
	g.save.recordFailure(g.difficulty)
	g.save.Continue = nil
	_ = g.save.write()
	g.puzzle = puzzleEntry{}
}

func (g *game) finishSuccess() {
	g.stopClock()
	ms := g.currentElapsed().Milliseconds()
	g.solvedMs = ms
	g.solvedMistakes = g.board.mistakes
	g.save.recordSuccess(g.difficulty, ms, g.board.mistakes)
	g.save.markCompleted(g.difficulty, g.puzzle.ID, g.board.mistakes, ms)
	g.save.Continue = nil
	_ = g.save.write()
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
