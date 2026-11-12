package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

const (
	viewMenu = iota
	viewSettings
	viewName
	viewJoin
	viewLobby
	viewPlay
	viewGuide
	viewStats
)

const (
	overlayNone = iota
	overlayLog
	overlayHistory
	overlayResign
	overlayLeave
)

const (
	menuAI = iota
	menuLocal
	menuHost
	menuJoin
	menuGuide
	menuStats
	menuName
	menuOptions
	menuQuit
	menuCount
)

const (
	rowDifficulty = iota
	rowPlayer2
	rowRounds
	rowTimer
	rowPort
	rowBindIP
	rowRoomCode
	rowDestroy
	rowDice
	rowEnd
	rowStart
	rowTheme
	rowSound
	rowColorblind
	rowWinMeter
	rowDone
)

const (
	setupHost    = "host"
	setupOptions = "options"
	choiceMenu   = "menu"
	rollDuration = 450 * time.Millisecond
	aiDelay      = 950 * time.Millisecond
	maxAddrLen   = 64
)

var roundChoices = []int{1, 3, 5, 0}
var timerChoices = []int{15, 30, 60, 0}

// controller is how the play screen drives a game, whether the session
// lives here (vs AI, same keyboard), on this host, or on a remote host.
type controller interface {
	state() *session
	seat() int // -1 when both players share this keyboard
	place(mv Move)
	aim(col, die int) // online: share live column/die aim with the opponent
	vote(v string)
	resign()
	leave()
	tick(now time.Time) bool
	netEvents() <-chan netEvent
	handle(ev netEvent, now time.Time) bool
	turnLeft(now time.Time) time.Duration
	disconnectLeft(now time.Time) (int, time.Duration)
	opponentGone() bool
	fairness() int
}

// lobby is the extra surface online controllers offer before a game starts.
type lobby interface {
	waitingSince() time.Time
	lobbyError() string
	retryConnect()
	hostPort() int
	hostRoomCode() string
}

type aiResult struct {
	key [2]int
	mv  Move
}

type offlineCtl struct {
	s        *session
	rng      *rand.Rand
	ai       bool
	aiKey    [2]int
	aiAt     time.Time
	aiCh     <-chan aiResult
	aiCancel context.CancelFunc
	syncAI   bool
}

func (c *offlineCtl) cancelAI() {
	if c.aiCancel != nil {
		c.aiCancel()
		c.aiCancel = nil
	}
	c.aiCh = nil
}

func (c *offlineCtl) state() *session { return c.s }

func (c *offlineCtl) seat() int {
	if c.ai {
		return 0
	}
	return -1
}

func (c *offlineCtl) place(mv Move) {
	if c.ai && c.s.Match.Turn != 0 {
		return
	}
	_ = c.s.place(c.s.Match.Turn, mv, c.rng)
}

func (c *offlineCtl) aim(int, int) {}

func (c *offlineCtl) vote(v string) { c.s.vote(0, v, c.rng) }

func (c *offlineCtl) resign() {
	c.cancelAI()
	switch {
	case c.s.Phase == phasePlay && c.ai:
		c.s.forfeitBy(0, reasonResign)
	case c.s.Phase == phasePlay:
		c.s.forfeitBy(c.s.Match.Turn, reasonResign)
	case c.s.Phase == phaseRoundOver && c.ai:
		c.s.forfeitBy(0, reasonResign)
	}
}

func (c *offlineCtl) leave() { c.cancelAI() }

func (c *offlineCtl) tick(now time.Time) bool {
	if !c.ai || c.s.Phase != phasePlay || c.s.Match.Turn != 1 {
		return false
	}
	key := [2]int{c.s.Series*1000 + c.s.Round, c.s.Match.Moves + 1}
	if key != c.aiKey {
		c.cancelAI()
		c.aiKey = key
		c.aiAt = now.Add(aiDelay)
		if c.s.Difficulty == diffHard && !c.syncAI {
			ctx, cancel := context.WithCancel(context.Background())
			c.aiCancel = cancel
			snap := c.s.Match
			subRNG := rand.New(rand.NewPCG(c.rng.Uint64(), c.rng.Uint64()))
			ch := make(chan aiResult, 1)
			c.aiCh = ch
			go func() {
				if mv, ok := chooseMoveContext(ctx, &snap, diffHard, subRNG); ok {
					ch <- aiResult{key: key, mv: mv}
				}
			}()
		}
		return false
	}
	if now.Before(c.aiAt) {
		return false
	}
	if c.aiCh != nil {
		select {
		case res := <-c.aiCh:
			c.aiCh = nil
			c.aiCancel = nil
			if res.key == key {
				_ = c.s.place(1, res.mv, c.rng)
				return true
			}
		default:
			return false
		}
	}
	_ = c.s.place(1, chooseMove(&c.s.Match, c.s.Difficulty, c.rng), c.rng)
	return true
}

func (c *offlineCtl) netEvents() <-chan netEvent                    { return nil }
func (c *offlineCtl) handle(netEvent, time.Time) bool               { return false }
func (c *offlineCtl) turnLeft(time.Time) time.Duration              { return 0 }
func (c *offlineCtl) disconnectLeft(time.Time) (int, time.Duration) { return -1, 0 }
func (c *offlineCtl) opponentGone() bool                            { return false }
func (c *offlineCtl) fairness() int                                 { return fairNone }

type game struct {
	screen   tcell.Screen
	width    int
	height   int
	pal      palette
	save     *saveData
	savePath string
	rng      *rand.Rand

	view      int
	menuIndex int
	setupMode string
	setIndex  int
	note      string
	input     string
	inputPos  int // caret index in runes for input / Player 2 name / Room code

	hostRoomCode string // ephemeral host room code, never persisted to disk
	joinCode     string // ephemeral guest room code, never persisted to disk
	joinField    int    // 0 = address, 1 = room code

	ctl           controller
	overlay       int
	overlayScroll int
	confirmIndex  int
	resultIndex   int
	hideResult    bool
	cursor        int
	dieSel        int
	guideScroll   int

	rollKey    [2]int
	rollAt     time.Time
	aimPub     [2]int // last online aim published as {col, die}; {-1,-1} = none
	effectKey  [2]int
	effectAt   time.Time
	effectLast Placement

	recRound      [2]int
	recSeries     int
	recordNote    string
	soundPhaseKey [2]int
	soundQueue    []soundID
	winCache      winProbCache

	replayActive bool
	replayStep   int
	replayFrames []replayFrame
}

func main() {
	f, err := parseFlags(os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}
	terminal := launchedFromTerminal()
	mode, err := chooseMode(f.gui, f.tui, terminal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "knucklebones: %v\n", err)
		os.Exit(2)
	}
	if mode == modeGUI && !guiAvailable {
		if f.gui {
			fmt.Fprintln(os.Stderr, "knucklebones: the GUI is not available in this build; run without -gui")
			os.Exit(1)
		}
		mode = modeTUI
	}
	// Ebitengine's package init is skipped on SSH/-tui (see ebiten_boot.go).
	// Still refuse RunGame in blocked sessions if init was not skipped.
	if mode == modeGUI && guiBlockedSession() {
		if f.gui {
			fmt.Fprintln(os.Stderr, "knucklebones: the GUI needs a local desktop session; over SSH use -tui")
			os.Exit(1)
		}
		mode = modeTUI
	}
	if mode == modeGUI {
		if !terminal {
			detachConsole()
		}
		if err := runGUI(); err != nil {
			fmt.Fprintf(os.Stderr, "knucklebones: %v\n", err)
			os.Exit(1)
		}
		return
	}
	ensureConsole()
	runTUI()
}

func newGame() *game {
	path := resolveSavePath()
	g := &game{
		savePath: path,
		save:     loadSave(path),
		rng:      rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(os.Getpid()))),
	}
	g.pal = paletteFor(g.save.Theme)
	return g
}

func (g *game) emitSound(id soundID) {
	if !g.save.Settings.Sound || id == sndNone || len(g.soundQueue) >= 4 {
		return
	}
	g.soundQueue = append(g.soundQueue, id)
}

func (g *game) drainSounds() []soundID {
	if len(g.soundQueue) == 0 {
		return nil
	}
	out := g.soundQueue
	g.soundQueue = nil
	return out
}

func (g *game) enterReplay(s *session) {
	frames := buildReplay(s)
	if len(frames) == 0 {
		return
	}
	g.replayFrames = frames
	g.replayStep = len(frames) - 1
	g.replayActive = true
	g.overlay = overlayNone
	g.emitSound(sndClick)
}

func (g *game) exitReplay() {
	g.replayActive = false
	g.replayStep = 0
	g.replayFrames = nil
}

func (g *game) afterChange() {
	g.syncView()
	g.checkRecord()
	g.syncAim()
	if g.screen != nil {
		g.render()
	}
}

func (g *game) persist() { _ = g.save.write(g.savePath) }

func (g *game) syncView() {
	if g.ctl == nil {
		return
	}
	s := g.ctl.state()
	if s == nil {
		return
	}
	if g.view == viewLobby {
		g.view = viewPlay
		g.overlay = overlayNone
	}
	key := [2]int{s.Series*1000 + s.Round, s.Match.Moves + 1}
	if s.Last != nil && s.Match.Moves > 0 && key != g.effectKey {
		g.effectKey = key
		g.effectAt = time.Now()
		g.effectLast = *s.Last
		if s.Last.Destroyed > 0 {
			g.emitSound(sndDestroy)
		} else {
			g.emitSound(sndPlace)
		}
	}
	if s.Phase != phasePlay {
		if g.overlay == overlayResign && s.Phase == phaseGameOver {
			g.overlay = overlayNone
		}
		if n := len(g.resultChoices(s)); g.resultIndex >= n {
			g.resultIndex = 0
		}
		phaseKey := [2]int{s.Series, s.Round}
		if phaseKey != g.soundPhaseKey {
			g.soundPhaseKey = phaseKey
			if s.Mode != modeLocal {
				w := s.Match.Winner()
				if s.Forfeit != nil {
					w = 1 - s.Forfeit.Loser
				}
				switch {
				case w == g.ctl.seat():
					g.emitSound(sndWin)
				case w >= 0:
					g.emitSound(sndLose)
				}
			}
		}
		return
	}
	if g.replayActive {
		g.exitReplay()
	}
	g.hideResult = false
	if key != g.rollKey {
		g.rollKey = key
		g.rollAt = time.Now()
		g.dieSel = 0
		g.resultIndex = 0
		g.recordNote = ""
		g.aimPub = [2]int{-1, -1}
		grid := &s.Match.Grids[s.Match.Turn]
		for i := 0; i < gridCols && !grid.Open(g.cursor); i++ {
			g.cursor = (g.cursor + 1) % gridCols
		}
		if s.Match.Moves == 0 && g.myTurn(s) {
			g.emitSound(sndTurn)
		}
	}
}

func (g *game) rolling() bool { return time.Since(g.rollAt) < rollDuration }

func (g *game) myTurn(s *session) bool {
	if s == nil || s.Phase != phasePlay {
		return false
	}
	seat := g.ctl.seat()
	return seat < 0 || s.Match.Turn == seat
}

// syncAim publishes the local column/die choice so an online opponent can
// watch the same placement preview while this player aims.
func (g *game) syncAim() {
	if g.ctl == nil {
		return
	}
	s := g.ctl.state()
	if s == nil || s.Mode != modeOnline || s.Phase != phasePlay {
		g.aimPub = [2]int{-1, -1}
		return
	}
	if !g.myTurn(s) || g.rolling() || g.overlay != overlayNone {
		return
	}
	key := [2]int{g.cursor, g.dieSel}
	if key == g.aimPub {
		return
	}
	g.aimPub = key
	g.ctl.aim(g.cursor, g.dieSel)
}

// previewMove is the column/die currently being aimed for the board preview.
// Local turns use the cursor; online opponent turns use the shared aim.
func (g *game) previewMove(s *session) (Move, bool) {
	if s == nil || s.Phase != phasePlay || g.rolling() || g.overlay != overlayNone {
		return Move{}, false
	}
	if g.myTurn(s) {
		return Move{Col: g.cursor, Die: g.dieSel}, true
	}
	if s.Mode == modeOnline && s.Aiming {
		return Move{Col: s.AimCol, Die: s.AimDie}, true
	}
	return Move{}, false
}

func (g *game) beginGame(c controller) {
	g.ctl = c
	g.overlay = overlayNone
	g.resultIndex = 0
	g.cursor = 1
	g.dieSel = 0
	g.rollKey = [2]int{}
	g.effectKey = [2]int{}
	g.effectAt = time.Time{}
	g.aimPub = [2]int{-1, -1}
	g.recRound = [2]int{}
	g.recSeries = 0
	g.soundPhaseKey = [2]int{}
	g.winCache = winProbCache{}
	g.exitReplay()
	g.recordNote = ""
	g.hideResult = false
	if c.state() != nil {
		g.view = viewPlay
	} else {
		g.view = viewLobby
	}
}

func (g *game) leaveGame() {
	if g.ctl != nil {
		g.ctl.leave()
		g.ctl = nil
	}
	g.overlay = overlayNone
	g.exitReplay()
	g.soundQueue = nil
	if g.view == viewPlay || g.view == viewLobby {
		g.view = viewMenu
	}
}

// ---- results and stats ----

func (g *game) checkRecord() {
	if g.ctl == nil {
		return
	}
	s := g.ctl.state()
	if s == nil || s.Phase == phasePlay {
		return
	}
	key := [2]int{s.Series, s.Round}
	if s.Forfeit == nil && key != g.recRound {
		g.recRound = key
		g.recordRound(s)
	}
	if s.Phase == phaseGameOver && s.Series != g.recSeries {
		g.recSeries = s.Series
		if s.Forfeit != nil && key != g.recRound {
			g.recRound = key
			g.recordForfeit(s)
		}
		g.recordSeries(s)
	}
}

func (g *game) recordRound(s *session) {
	r := g.save.recordFor(recordKey(s.Mode, s.Difficulty))
	r.Games++
	sc := s.Match.Scores()
	w := s.Match.Winner()
	g.recordNote = ""
	if s.Mode == modeLocal {
		if w < 0 {
			r.Draws++
		}
		for p := 0; p < 2; p++ {
			if sc[p] > r.Best {
				r.Best = sc[p]
			}
			g.save.addHighScore(highScore{Name: s.Names[p], Score: sc[p], Opponent: s.Names[1-p], Against: sc[1-p], Mode: s.Mode})
		}
	} else {
		me := g.ctl.seat()
		switch {
		case w < 0:
			r.Draws++
		case w == me:
			r.Wins++
		default:
			r.Losses++
		}
		if sc[me] > r.Best {
			r.Best = sc[me]
		}
		if rank := g.save.addHighScore(highScore{Name: s.Names[me], Score: sc[me], Opponent: s.Names[1-me], Against: sc[1-me], Mode: s.Mode}); rank > 0 {
			g.recordNote = fmt.Sprintf("New high score! #%d on your Top 10", rank)
		}
	}
	g.persist()
}

func (g *game) recordForfeit(s *session) {
	r := g.save.recordFor(recordKey(s.Mode, s.Difficulty))
	r.Games++
	if s.Mode != modeLocal {
		if s.Forfeit.Loser == g.ctl.seat() {
			r.Losses++
		} else {
			r.Wins++
		}
	}
	g.persist()
}

func (g *game) recordSeries(s *session) {
	if s.Mode == modeLocal || s.BestOf < 3 || s.FreePlay {
		return
	}
	r := g.save.recordFor(recordKey(s.Mode, s.Difficulty))
	switch w := s.seriesWinner(); {
	case w == g.ctl.seat():
		r.SeriesWins++
	case w >= 0:
		r.SeriesLosses++
	}
	g.persist()
}

// resultChoices are the buttons under a finished round.
func (g *game) resultChoices(s *session) []string {
	var out []string
	if !g.ctl.opponentGone() {
		out = append(out, s.options()...)
	}
	return append(out, choiceMenu)
}

// ---- input ----

func isRune(e *tcell.EventKey, rs ...rune) bool {
	if e.Key() != tcell.KeyRune {
		return false
	}
	r := unicode.ToLower(e.Rune())
	for _, x := range rs {
		if r == x {
			return true
		}
	}
	return false
}

func isUp(e *tcell.EventKey) bool    { return e.Key() == tcell.KeyUp || isRune(e, 'w') }
func isDown(e *tcell.EventKey) bool  { return e.Key() == tcell.KeyDown || isRune(e, 's') }
func isLeft(e *tcell.EventKey) bool  { return e.Key() == tcell.KeyLeft || isRune(e, 'a') }
func isRight(e *tcell.EventKey) bool { return e.Key() == tcell.KeyRight || isRune(e, 'd') }
func isEnter(e *tcell.EventKey) bool { return e.Key() == tcell.KeyEnter }
func isBack(e *tcell.EventKey) bool {
	return e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyBackspace2 || e.Key() == tcell.KeyDelete
}

func (g *game) handleKey(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyCtrlC {
		return true
	}
	switch g.view {
	case viewMenu:
		return g.keyMenu(e)
	case viewSettings:
		g.keySettings(e)
	case viewName:
		g.keyName(e)
	case viewJoin:
		g.keyJoin(e)
	case viewLobby:
		g.keyLobby(e)
	case viewPlay:
		g.keyPlay(e)
	case viewGuide:
		g.keyGuide(e)
	case viewStats:
		if e.Key() == tcell.KeyEscape || isEnter(e) || isRune(e, 'q') {
			g.view = viewMenu
		}
	}
	return false
}

func (g *game) toggleTheme() {
	g.stepTheme(1)
}

func (g *game) stepTheme(step int) {
	idx := 0
	for i, t := range themeOrder {
		if t == g.save.Theme {
			idx = i
		}
	}
	g.save.Theme = themeOrder[(idx+step+len(themeOrder))%len(themeOrder)]
	g.pal = paletteFor(g.save.Theme)
	g.persist()
}

func (g *game) keyMenu(e *tcell.EventKey) bool {
	switch {
	case e.Key() == tcell.KeyEscape:
		return true
	case isUp(e):
		g.menuIndex = (g.menuIndex + menuCount - 1) % menuCount
	case isDown(e):
		g.menuIndex = (g.menuIndex + 1) % menuCount
	case isRune(e, 't'):
		g.toggleTheme()
		g.emitSound(sndClick)
	case g.menuIndex == menuOptions && isLeft(e):
		g.stepTheme(-1)
		g.emitSound(sndClick)
	case g.menuIndex == menuOptions && isRight(e):
		g.stepTheme(1)
		g.emitSound(sndClick)
	case isEnter(e) || isRune(e, ' '):
		g.note = ""
		g.emitSound(sndClick)
		switch g.menuIndex {
		case menuAI:
			g.openSettings(modeAI)
		case menuLocal:
			g.openSettings(modeLocal)
		case menuHost:
			g.openSettings(setupHost)
		case menuJoin:
			g.input = g.save.Settings.JoinAddr
			g.joinCode = ""
			g.joinField = 0
			g.inputPos = runeLen(g.input)
			g.view = viewJoin
		case menuGuide:
			g.guideScroll = 0
			g.view = viewGuide
		case menuStats:
			g.view = viewStats
		case menuName:
			g.input = g.save.Name
			g.inputPos = runeLen(g.input)
			g.view = viewName
		case menuOptions:
			g.openSettings(setupOptions)
		case menuQuit:
			return true
		}
	}
	return false
}

func (g *game) openSettings(mode string) {
	g.setupMode = mode
	if mode == setupOptions {
		g.setIndex = 0
	} else {
		g.setIndex = len(g.settingRows()) - 1
	}
	g.focusSettingCaret()
	g.view = viewSettings
}

func (g *game) settingRows() []int {
	switch g.setupMode {
	case modeAI:
		return []int{rowDifficulty, rowRounds, rowDestroy, rowDice, rowEnd, rowStart}
	case modeLocal:
		return []int{rowPlayer2, rowRounds, rowDestroy, rowDice, rowEnd, rowStart}
	case setupOptions:
		return []int{rowTheme, rowSound, rowColorblind, rowWinMeter, rowDone}
	}
	return []int{rowRounds, rowTimer, rowPort, rowBindIP, rowRoomCode, rowDestroy, rowDice, rowEnd, rowStart}
}

func cycleInt(list []int, cur, step int) int {
	idx := 0
	for i, v := range list {
		if v == cur {
			idx = i
		}
	}
	return list[(idx+step+len(list))%len(list)]
}

func cycleString(list []string, cur string, step int) string {
	idx := 0
	for i, v := range list {
		if v == cur {
			idx = i
		}
	}
	return list[(idx+step+len(list))%len(list)]
}

func (g *game) changeSetting(row, step int) {
	st := &g.save.Settings
	switch row {
	case rowDifficulty:
		st.Difficulty = cycleString(difficultyOrder, st.Difficulty, step)
	case rowRounds:
		st.BestOf = cycleInt(roundChoices, st.BestOf, step)
	case rowTimer:
		st.TurnSeconds = cycleInt(timerChoices, st.TurnSeconds, step)
	case rowBindIP:
		st.BindIP = cycleString(bindIPChoices(), st.BindIP, step)
	case rowDestroy:
		st.Rules.NoDestroy = !st.Rules.NoDestroy
	case rowDice:
		st.Rules.TwoDice = !st.Rules.TwoDice
	case rowEnd:
		st.Rules.FillBoth = !st.Rules.FillBoth
	case rowTheme:
		g.stepTheme(step)
		g.emitSound(sndClick)
		return
	case rowSound:
		st.Sound = !st.Sound
		if st.Sound {
			g.emitSound(sndClick)
		}
	case rowColorblind:
		st.Colorblind = !st.Colorblind
	case rowWinMeter:
		st.WinMeter = !st.WinMeter
	default:
		return
	}
	g.emitSound(sndClick)
	g.persist()
}

func isRoomCodeRune(r rune) bool {
	return unicode.IsPrint(r) && !unicode.IsSpace(r) && !unicode.IsControl(r) && !isBidiControl(r)
}

func (g *game) keySettings(e *tcell.EventKey) {
	rows := g.settingRows()
	row := rows[g.setIndex]
	if row == rowPlayer2 {
		if next, pos, ok := editLine(e, g.save.Settings.Player2, g.inputPos, maxNameLen, unicode.IsPrint); ok {
			g.save.Settings.Player2 = next
			g.inputPos = pos
			return
		}
	}
	if row == rowPort {
		if next, pos, ok := editLine(e, g.portEditString(), g.inputPos, 5, func(r rune) bool {
			return r >= '0' && r <= '9'
		}); ok {
			g.applyPortEdit(next)
			g.inputPos = pos
			return
		}
	}
	if row == rowRoomCode {
		if next, pos, ok := editLine(e, g.hostRoomCode, g.inputPos, maxRoomCodeLen, isRoomCodeRune); ok {
			g.hostRoomCode = next
			g.inputPos = pos
			return
		}
	}
	switch {
	case e.Key() == tcell.KeyEscape:
		g.save.Settings.Player2 = cleanName(g.save.Settings.Player2, "Player 2")
		g.hostRoomCode = cleanRoomCode(g.hostRoomCode)
		g.persist()
		g.view = viewMenu
	case e.Key() == tcell.KeyUp || e.Key() == tcell.KeyBacktab:
		g.setIndex = (g.setIndex + len(rows) - 1) % len(rows)
		g.focusSettingCaret()
	case e.Key() == tcell.KeyDown || e.Key() == tcell.KeyTab:
		g.setIndex = (g.setIndex + 1) % len(rows)
		g.focusSettingCaret()
	case e.Key() == tcell.KeyLeft:
		g.changeSetting(row, -1)
	case e.Key() == tcell.KeyRight:
		g.changeSetting(row, 1)
	case isUp(e):
		g.setIndex = (g.setIndex + len(rows) - 1) % len(rows)
		g.focusSettingCaret()
	case isDown(e):
		g.setIndex = (g.setIndex + 1) % len(rows)
		g.focusSettingCaret()
	case isLeft(e):
		g.changeSetting(row, -1)
	case isRight(e):
		g.changeSetting(row, 1)
	case isEnter(e) || isRune(e, ' '):
		switch row {
		case rowStart:
			g.startFromSettings()
		case rowDone:
			g.emitSound(sndClick)
			g.view = viewMenu
		case rowPlayer2, rowPort, rowRoomCode:
			g.setIndex = (g.setIndex + 1) % len(rows)
			g.focusSettingCaret()
		default:
			g.changeSetting(row, 1)
		}
	}
}

// portEditString is the digits shown/edited in the Port field.
func (g *game) portEditString() string {
	if g.save.Settings.Port <= 0 {
		return ""
	}
	return strconv.Itoa(g.save.Settings.Port)
}

func (g *game) applyPortEdit(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		g.save.Settings.Port = 0
		return
	}
	p, err := strconv.Atoi(s)
	if err != nil || p < 0 || p > 65535 {
		return
	}
	g.save.Settings.Port = p
}

func (g *game) focusSettingCaret() {
	rows := g.settingRows()
	if g.setIndex < 0 || g.setIndex >= len(rows) {
		return
	}
	switch rows[g.setIndex] {
	case rowPlayer2:
		g.inputPos = runeLen(g.save.Settings.Player2)
	case rowPort:
		if g.save.Settings.Port <= 0 {
			g.save.Settings.Port = defaultPort
		}
		g.inputPos = runeLen(g.portEditString())
	case rowRoomCode:
		g.inputPos = runeLen(g.hostRoomCode)
	}
}

func (g *game) startFromSettings() {
	st := &g.save.Settings
	st.Player2 = cleanName(st.Player2, "Player 2")
	g.hostRoomCode = cleanRoomCode(g.hostRoomCode)
	g.note = ""
	switch g.setupMode {
	case modeAI:
		names := [2]string{g.save.Name, aiName[st.Difficulty]}
		g.beginGame(&offlineCtl{s: newSession(modeAI, names, st.Difficulty, st.BestOf, st.Rules, g.rng), rng: g.rng, ai: true})
	case modeLocal:
		p2 := st.Player2
		if strings.EqualFold(p2, g.save.Name) {
			p2 += " (2)"
		}
		names := [2]string{g.save.Name, p2}
		g.beginGame(&offlineCtl{s: newSession(modeLocal, names, "", st.BestOf, st.Rules, g.rng), rng: g.rng})
	case setupHost:
		if st.Port <= 0 {
			st.Port = defaultPort
		}
		h, err := startHost(g.save.Name, st.Port, hostConfig{
			BestOf:      st.BestOf,
			Rules:       st.Rules,
			TurnSeconds: st.TurnSeconds,
			BindIP:      st.BindIP,
			RoomCode:    g.hostRoomCode,
		}, g.rng)
		if err != nil {
			g.note = fmt.Sprintf("Could not open port %d: %v", st.Port, err)
			return
		}
		g.beginGame(h)
	}
	g.persist()
}

func runeLen(s string) int { return len([]rune(s)) }

func clampRunePos(s string, pos int) int {
	n := runeLen(s)
	if pos < 0 {
		return 0
	}
	if pos > n {
		return n
	}
	return pos
}

// editLine applies a text-editing key at the caret. Arrow/Home/End move the
// caret; Backspace/Delete edit around it. ok is false for unrelated keys.
// Left/Right use the key codes only so typed a/d still insert.
func editLine(e *tcell.EventKey, s string, pos, max int, allow func(rune) bool) (string, int, bool) {
	r := []rune(s)
	pos = clampRunePos(s, pos)
	switch {
	case e.Key() == tcell.KeyLeft:
		if pos > 0 {
			pos--
		}
		return s, pos, true
	case e.Key() == tcell.KeyRight:
		if pos < len(r) {
			pos++
		}
		return s, pos, true
	case e.Key() == tcell.KeyHome:
		return s, 0, true
	case e.Key() == tcell.KeyEnd:
		return s, len(r), true
	case e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyBackspace2:
		if pos > 0 {
			r = append(r[:pos-1], r[pos:]...)
			pos--
			return string(r), pos, true
		}
		return s, pos, true
	case e.Key() == tcell.KeyDelete:
		if pos < len(r) {
			r = append(r[:pos], r[pos+1:]...)
			return string(r), pos, true
		}
		return s, pos, true
	case e.Key() == tcell.KeyCtrlU:
		return "", 0, true
	case e.Key() == tcell.KeyRune && allow(e.Rune()):
		if len(r) < max {
			ins := []rune{e.Rune()}
			r = append(r[:pos], append(ins, r[pos:]...)...)
			pos++
			return string(r), pos, true
		}
		return s, pos, true
	}
	return s, pos, false
}

// normalizeJoinAddr adds :8647 when the host port is missing or blank.
func normalizeJoinAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if strings.TrimSpace(port) != "" {
			return net.JoinHostPort(host, port)
		}
		if strings.TrimSpace(host) == "" {
			return addr
		}
		return net.JoinHostPort(host, strconv.Itoa(defaultPort))
	}
	if strings.HasSuffix(addr, ":") {
		host := strings.TrimSpace(strings.TrimSuffix(addr, ":"))
		if host == "" {
			return addr
		}
		return net.JoinHostPort(host, strconv.Itoa(defaultPort))
	}
	if strings.Contains(addr, ":") {
		return addr
	}
	return net.JoinHostPort(addr, strconv.Itoa(defaultPort))
}

func (g *game) keyName(e *tcell.EventKey) {
	switch {
	case e.Key() == tcell.KeyEscape:
		g.view = viewMenu
	case isEnter(e):
		g.save.Name = cleanName(g.input, g.save.Name)
		g.persist()
		g.view = viewMenu
	default:
		g.input, g.inputPos, _ = editLine(e, g.input, g.inputPos, maxNameLen, unicode.IsPrint)
	}
}

func (g *game) focusJoinField(idx int) {
	if idx < 0 || idx > 1 {
		idx = 0
	}
	g.joinField = idx
	if idx == 0 {
		g.inputPos = runeLen(g.input)
	} else {
		g.inputPos = runeLen(g.joinCode)
	}
}

func (g *game) keyJoin(e *tcell.EventKey) {
	switch {
	case e.Key() == tcell.KeyEscape:
		g.view = viewMenu
	case e.Key() == tcell.KeyUp || e.Key() == tcell.KeyBacktab || e.Key() == tcell.KeyDown || e.Key() == tcell.KeyTab:
		g.focusJoinField(1 - g.joinField)
	case isEnter(e):
		addr := normalizeJoinAddr(g.input)
		if addr == "" {
			g.note = "Enter the host's address, for example 192.168.1.20:8647"
			g.focusJoinField(0)
			return
		}
		code := cleanRoomCode(g.joinCode)
		g.input = addr
		g.joinCode = code
		g.inputPos = runeLen(addr)
		g.save.Settings.JoinAddr = addr
		g.persist()
		g.note = ""
		g.beginGame(startClientWithCode(g.save.Name, addr, code))
	default:
		if g.joinField == 1 {
			g.joinCode, g.inputPos, _ = editLine(e, g.joinCode, g.inputPos, maxRoomCodeLen, isRoomCodeRune)
		} else {
			g.input, g.inputPos, _ = editLine(e, g.input, g.inputPos, maxAddrLen, func(r rune) bool { return unicode.IsPrint(r) && r != ' ' })
		}
	}
}

func (g *game) keyLobby(e *tcell.EventKey) {
	switch {
	case e.Key() == tcell.KeyEscape:
		g.leaveGame()
	case isEnter(e):
		if l, ok := g.ctl.(lobby); ok && l.lobbyError() != "" {
			l.retryConnect()
		}
	}
}

func (g *game) keyGuide(e *tcell.EventKey) {
	switch {
	case e.Key() == tcell.KeyEscape || isEnter(e) || isRune(e, 'q'):
		g.view = viewMenu
	case isUp(e):
		g.guideScroll--
	case isDown(e):
		g.guideScroll++
	case e.Key() == tcell.KeyPgUp:
		g.guideScroll -= g.height - 4
	case e.Key() == tcell.KeyPgDn || isRune(e, ' '):
		g.guideScroll += g.height - 4
	case e.Key() == tcell.KeyHome:
		g.guideScroll = 0
	}
	if g.guideScroll < 0 {
		g.guideScroll = 0
	}
}

func (g *game) canResign(s *session) bool {
	switch s.Phase {
	case phasePlay:
		return g.ctl.seat() >= 0 || s.Mode == modeLocal
	case phaseRoundOver:
		return s.Mode != modeLocal
	}
	return false
}

func (g *game) keyPlay(e *tcell.EventKey) {
	s := g.ctl.state()
	if s == nil {
		return
	}
	switch g.overlay {
	case overlayResign, overlayLeave:
		switch {
		case isLeft(e) || isRight(e) || e.Key() == tcell.KeyTab:
			g.confirmIndex = 1 - g.confirmIndex
		case e.Key() == tcell.KeyEscape || isRune(e, 'n'):
			g.overlay = overlayNone
		case isEnter(e) || isRune(e, 'y'):
			if isRune(e, 'y') {
				g.confirmIndex = 1
			}
			which := g.overlay
			g.overlay = overlayNone
			if g.confirmIndex == 1 {
				if which == overlayResign {
					g.ctl.resign()
				} else {
					g.leaveGame()
				}
			}
		}
		return
	case overlayLog, overlayHistory:
		switch {
		case e.Key() == tcell.KeyEscape || isEnter(e):
			g.overlay = overlayNone
		case isRune(e, 'l'):
			g.toggleOverlay(overlayLog)
		case isRune(e, 'h'):
			g.toggleOverlay(overlayHistory)
		case isUp(e):
			g.overlayScroll++
		case isDown(e):
			g.overlayScroll--
		case e.Key() == tcell.KeyPgUp:
			g.overlayScroll += g.height - 6
		case e.Key() == tcell.KeyPgDn:
			g.overlayScroll -= g.height - 6
		}
		if g.overlayScroll < 0 {
			g.overlayScroll = 0
		}
		return
	}

	switch {
	case isRune(e, 'p'):
		g.save.Settings.WinMeter = !g.save.Settings.WinMeter
		g.emitSound(sndClick)
		g.persist()
		return
	case isRune(e, 'c'):
		g.save.Settings.Colorblind = !g.save.Settings.Colorblind
		g.emitSound(sndClick)
		g.persist()
		return
	case isRune(e, 's'):
		g.save.Settings.Sound = !g.save.Settings.Sound
		if g.save.Settings.Sound {
			g.emitSound(sndClick)
		}
		g.persist()
		return
	case isRune(e, 'l'):
		g.toggleOverlay(overlayLog)
		return
	case isRune(e, 'h'):
		g.toggleOverlay(overlayHistory)
		return
	case g.replayActive:
		switch {
		case e.Key() == tcell.KeyEscape || isEnter(e) || isRune(e, 'x'):
			g.exitReplay()
			g.emitSound(sndClick)
		case isLeft(e):
			if g.replayStep > 0 {
				g.replayStep--
				g.emitSound(sndClick)
			}
		case isRight(e):
			if g.replayStep+1 < len(g.replayFrames) {
				g.replayStep++
				g.emitSound(sndClick)
			}
		case e.Key() == tcell.KeyHome:
			g.replayStep = 0
		case e.Key() == tcell.KeyEnd && len(g.replayFrames) > 0:
			g.replayStep = len(g.replayFrames) - 1
		}
		return
	case isRune(e, 'r'):
		if g.canResign(s) {
			g.overlay = overlayResign
			g.confirmIndex = 0
		}
		return
	case e.Key() == tcell.KeyEscape:
		if s.Phase == phaseGameOver || g.ctl.opponentGone() {
			g.leaveGame()
		} else {
			g.overlay = overlayLeave
			g.confirmIndex = 0
		}
		return
	}

	if s.Phase != phasePlay {
		choices := g.resultChoices(s)
		switch {
		case isRune(e, 'v'):
			g.hideResult = !g.hideResult
			g.emitSound(sndClick)
		case isRune(e, 'x'):
			if canReplay(s) {
				g.enterReplay(s)
			}
		case isLeft(e):
			g.resultIndex = (g.resultIndex + len(choices) - 1) % len(choices)
		case isRight(e) || e.Key() == tcell.KeyTab:
			g.resultIndex = (g.resultIndex + 1) % len(choices)
		case isEnter(e) || isRune(e, ' '):
			g.emitSound(sndClick)
			switch c := choices[g.resultIndex]; c {
			case choiceMenu:
				g.leaveGame()
			default:
				g.ctl.vote(c)
			}
		}
		return
	}

	if !g.myTurn(s) || g.rolling() {
		return
	}
	switch {
	case isLeft(e):
		g.cursor = (g.cursor + gridCols - 1) % gridCols
	case isRight(e):
		g.cursor = (g.cursor + 1) % gridCols
	case e.Key() == tcell.KeyTab || isRune(e, 'q', 'e') || isUp(e) || isDown(e):
		if s.Match.Dice[1] != 0 {
			g.dieSel = 1 - g.dieSel
		}
	case e.Key() == tcell.KeyRune && e.Rune() >= '1' && e.Rune() <= '3':
		g.cursor = int(e.Rune() - '1')
		g.placeAtCursor(s)
	case isEnter(e) || isRune(e, ' '):
		g.placeAtCursor(s)
	}
}

func (g *game) toggleOverlay(o int) {
	if g.overlay == o {
		g.overlay = overlayNone
		return
	}
	g.overlay = o
	g.overlayScroll = 0
}

func (g *game) placeAtCursor(s *session) {
	mv := Move{Col: g.cursor, Die: g.dieSel}
	if !s.Match.Legal(mv) {
		return
	}
	g.ctl.place(mv)
}
