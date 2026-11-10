package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	dieW    = 7
	dieH    = 3
	colGap  = 2
	slotGap = 1
	gridW   = gridCols*dieW + (gridCols-1)*colGap
	gridH   = gridRows*dieH + (gridRows-1)*slotGap
	pipRune = '●'
)

// Pip layout for each face on a 3×3 lattice.
var pipRows = [7][3]string{
	{},
	{"...", ".o.", "..."},
	{"..o", "...", "o.."},
	{"..o", ".o.", "o.."},
	{"o.o", "...", "o.o"},
	{"o.o", ".o.", "o.o"},
	{"o.o", "o.o", "o.o"},
}

func (g *game) st(fg tcell.Color) tcell.Style {
	return tcell.StyleDefault.Foreground(fg).Background(g.pal.bg)
}

func (g *game) render() {
	s := g.screen
	s.Fill(' ', g.st(g.pal.fg))
	switch g.view {
	case viewMenu:
		g.drawMenu()
	case viewSettings:
		g.drawSettings()
	case viewName:
		g.drawPrompt("Your name", "Shown to your opponent and on your Top 10.", g.input)
	case viewJoin:
		g.drawPrompt("Join an online game", "Host address (host:port)", g.input)
	case viewLobby:
		g.drawLobby()
	case viewPlay:
		g.drawPlay()
	case viewGuide:
		g.drawGuide()
	case viewStats:
		g.drawStats()
	}
	s.Show()
}

// ---- drawing primitives ----

func drawText(s tcell.Screen, x, y int, text string, st tcell.Style) int {
	for _, r := range text {
		if r == '\uFE0F' || r == '\uFE0E' {
			continue
		}
		s.SetContent(x, y, r, nil, st)
		w := runewidth.RuneWidth(r)
		if w < 1 {
			w = 1
		}
		x += w
	}
	return x
}

func textWidth(s string) int { return runewidth.StringWidth(s) }

func drawCentered(s tcell.Screen, cx, y int, text string, st tcell.Style) {
	x := cx - textWidth(text)/2
	if x < 0 {
		x = 0
	}
	drawText(s, x, y, text, st)
}

func fillRect(s tcell.Screen, x, y, w, h int, st tcell.Style) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.SetContent(x+dx, y+dy, ' ', nil, st)
		}
	}
}

// drawOverlayBox blanks whole rows first so side panels do not peek out
// beside a dialog.
func (g *game) drawOverlayBox(x, y, w, h int, title string) {
	fillRect(g.screen, 0, y, g.width, h, g.st(g.pal.fg))
	g.drawBox(x, y, w, h, title)
}

func (g *game) drawBox(x, y, w, h int, title string) {
	st := g.st(g.pal.frame)
	fillRect(g.screen, x, y, w, h, g.st(g.pal.fg))
	for dx := 1; dx < w-1; dx++ {
		g.screen.SetContent(x+dx, y, '─', nil, st)
		g.screen.SetContent(x+dx, y+h-1, '─', nil, st)
	}
	for dy := 1; dy < h-1; dy++ {
		g.screen.SetContent(x, y+dy, '│', nil, st)
		g.screen.SetContent(x+w-1, y+dy, '│', nil, st)
	}
	g.screen.SetContent(x, y, '╭', nil, st)
	g.screen.SetContent(x+w-1, y, '╮', nil, st)
	g.screen.SetContent(x, y+h-1, '╰', nil, st)
	g.screen.SetContent(x+w-1, y+h-1, '╯', nil, st)
	if title != "" {
		drawCentered(g.screen, x+w/2, y, " "+title+" ", g.st(g.pal.accent).Bold(true))
	}
}

func wrap(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		cur := ""
		for _, w := range strings.Fields(para) {
			switch {
			case cur == "":
				cur = w
			case textWidth(cur)+1+textWidth(w) <= width:
				cur += " " + w
			default:
				lines = append(lines, cur)
				cur = w
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

func drawDie(s tcell.Screen, x, y, v int, bg, pip tcell.Color) {
	st := tcell.StyleDefault.Background(bg).Foreground(pip)
	fillRect(s, x, y, dieW, dieH, st)
	if v < 1 || v > 6 {
		return
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if pipRows[v][r][c] == 'o' {
				s.SetContent(x+1+c*2, y+r, pipRune, nil, st)
			}
		}
	}
}

func (g *game) dieColors(matches int) (tcell.Color, tcell.Color) {
	switch {
	case matches >= 3:
		return g.pal.tripleBg, g.pal.triPip
	case matches == 2:
		return g.pal.pairBg, g.pal.pairPip
	}
	return g.pal.dieBg, g.pal.pip
}

// rollingFace cycles faces while a fresh die tumbles.
func rollingFace(seed int) int {
	return int(time.Now().UnixMilli()/70+int64(seed*5))%6 + 1
}

// ---- names and messages ----

func (g *game) isYou(s *session, p int) bool {
	seat := g.ctl.seat()
	return seat >= 0 && p == seat && s.Mode != modeLocal
}

func (g *game) who(s *session, p int) string {
	if g.isYou(s, p) {
		return "You"
	}
	return s.Names[p]
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (g *game) forfeitText(s *session) string {
	f := s.Forfeit
	loser, winner := s.Names[f.Loser], s.Names[1-f.Loser]
	youWin := g.isYou(s, 1-f.Loser)
	youLose := g.isYou(s, f.Loser)
	switch f.Reason {
	case reasonDisconnect:
		switch {
		case youWin:
			return "Your opponent disconnected. You win!"
		case youLose:
			return "You disconnected. " + winner + " wins."
		}
		return loser + " disconnected. " + winner + " wins."
	case reasonTimeout:
		switch {
		case youWin:
			return "Your opponent timed out three times. You win!"
		case youLose:
			return "You timed out three times. " + winner + " wins."
		}
		return loser + " timed out three times. " + winner + " wins."
	}
	switch {
	case youWin:
		return "Your opponent resigned. You win!"
	case youLose:
		return "You resigned. " + winner + " wins."
	}
	return loser + " resigned. " + winner + " wins."
}

func (g *game) logText(s *session, e logEntry) string {
	switch e.Kind {
	case logFirst:
		verb := " goes first."
		if g.isYou(s, e.Player) {
			verb = " go first."
		}
		return fmt.Sprintf("Round %d · %s%s", e.Round, g.who(s, e.Player), verb)
	case logPlace:
		t := fmt.Sprintf("%s placed a %d in column %d (+%d)", g.who(s, e.Player), e.Die, e.Col+1, e.Gain)
		if e.Destroyed > 0 {
			t += fmt.Sprintf(", destroying %d %s (−%d)", e.Destroyed, plural(e.Destroyed, "die", "dice"), e.Lost)
		}
		return t + "."
	case logTimeout:
		return fmt.Sprintf("%s ran out of time (%d/%d).", g.who(s, e.Player), e.Count, maxTimeouts)
	case logRound:
		t := fmt.Sprintf("Round %d ended %s %d – %d %s.", e.Round, s.Names[0], e.Scores[0], e.Scores[1], s.Names[1])
		if e.Player < 0 {
			return t + " Tie."
		}
		return t + " " + g.who(s, e.Player) + " won."
	case logForfeit:
		saved := s.Forfeit
		s.Forfeit = &forfeit{Reason: e.Reason, Loser: e.Player}
		t := g.forfeitText(s)
		s.Forfeit = saved
		return t
	}
	return ""
}

// resultText is the headline for a finished round or game.
func (g *game) resultText(s *session) (string, tcell.Color) {
	if s.Forfeit != nil {
		c := g.pal.accent
		if g.isYou(s, 1-s.Forfeit.Loser) {
			c = g.pal.win
		} else if g.isYou(s, s.Forfeit.Loser) {
			c = g.pal.lose
		}
		return g.forfeitText(s), c
	}
	sc := s.Match.Scores()
	w := s.Match.Winner()
	scope := "round"
	if s.Phase == phaseGameOver {
		scope = "game"
		w = s.seriesWinner()
	}
	switch {
	case w < 0 && scope == "game":
		return "This is a tie! Nobody wins!", g.pal.accent
	case w < 0:
		return "This is a tie! Nobody wins this round!", g.pal.accent
	case g.isYou(s, w):
		return fmt.Sprintf("You won the %s with %d points!", scope, sc[w]), g.pal.win
	}
	c := g.pal.accent
	if g.ctl.seat() >= 0 && s.Mode != modeLocal {
		c = g.pal.lose
	}
	return fmt.Sprintf("%s won the %s with %d points!", s.Names[w], scope, sc[w]), c
}

func choiceLabel(c string) string {
	switch c {
	case voteContinue:
		return "Continue game"
	case voteRematch:
		return "Play another game"
	case voteFree:
		return "Go into free play"
	}
	return "Menu"
}

// ---- menu and setup screens ----

func (g *game) drawTitle(y int) {
	drawCentered(g.screen, g.width/2, y, "K N U C K L E B O N E S", g.st(g.pal.title).Bold(true))
}

func (g *game) drawMenu() {
	w, h := g.width, g.height
	top := 0
	if h > 26 {
		top = (h - 26) / 2
	}
	g.drawTitle(top + 1)
	dx := w/2 - (6*dieW+5)/2
	for v := 1; v <= 6; v++ {
		drawDie(g.screen, dx+(v-1)*(dieW+1), top+3, v, g.pal.dieBg, g.pal.pip)
	}
	drawCentered(g.screen, w/2, top+7, "The dice game from Cult of the Lamb", g.st(g.pal.dim))

	labels := [menuCount]string{
		menuAI:    "Play against an AI",
		menuLocal: "Play against someone (same keyboard)",
		menuHost:  "Host an online game",
		menuJoin:  "Join an online game",
		menuGuide: "How to play",
		menuStats: "Stats",
		menuName:  "Name: " + g.save.Name,
		menuTheme: "Theme: " + map[string]string{themeDark: "Dark", themeLight: "Light"}[g.save.Theme],
		menuQuit:  "Quit",
	}
	y := top + 9
	for i, l := range labels {
		st := g.st(g.pal.fg)
		if i == g.menuIndex {
			st = g.st(g.pal.accent).Bold(true)
			l = "▶ " + l + " ◀"
		}
		drawCentered(g.screen, w/2, y+i, l, st)
	}
	drawCentered(g.screen, w/2, h-4, "Knucklebones is from Cult of the Lamb by Massive Monster.", g.st(g.pal.dim))
	drawCentered(g.screen, w/2, h-3, "This fan-made version is not an official implementation.", g.st(g.pal.dim))
	drawCentered(g.screen, w/2, h-1, "↑↓ select · Enter · T theme · Esc quit", g.st(g.pal.dim))
}

func onOff(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}

func (g *game) settingText(row int) (string, string) {
	st := g.save.Settings
	switch row {
	case rowDifficulty:
		return "AI difficulty", difficultyLabel[st.Difficulty] + " · " + aiName[st.Difficulty]
	case rowPlayer2:
		return "Player 2 name", st.Player2
	case rowRounds:
		return "Number of rounds", roundsLabel(st.BestOf)
	case rowTimer:
		if st.TurnSeconds == 0 {
			return "Turn timer", "Off"
		}
		return "Turn timer", fmt.Sprintf("%d seconds", st.TurnSeconds)
	case rowPort:
		if st.Port == 0 {
			return "Port", ""
		}
		return "Port", fmt.Sprint(st.Port)
	case rowDestroy:
		return "Destroy matching dice", onOff(!st.Rules.NoDestroy)
	case rowDice:
		if st.Rules.TwoDice {
			return "Dice per turn", "Two (place one)"
		}
		return "Dice per turn", "One"
	case rowEnd:
		if st.Rules.FillBoth {
			return "Game ends when", "Both boards are full"
		}
		return "Game ends when", "One board is full"
	}
	return "Start game", ""
}

func (g *game) drawSettings() {
	w, h := g.width, g.height
	titles := map[string]string{modeAI: "Play against an AI", modeLocal: "Play against someone", setupHost: "Host an online game"}
	g.drawTitle(1)
	drawCentered(g.screen, w/2, 3, titles[g.setupMode], g.st(g.pal.fg).Bold(true))
	drawCentered(g.screen, w/2, 4, "Game settings", g.st(g.pal.dim))

	rows := g.settingRows()
	y := 6
	lx := w/2 - 24
	if lx < 1 {
		lx = 1
	}
	for i, row := range rows {
		label, value := g.settingText(row)
		sel := i == g.setIndex
		if row == rowStart {
			st := g.st(g.pal.fg)
			text := "  Start game  "
			if sel {
				st = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
				text = "▶ Start game ◀"
			}
			drawCentered(g.screen, w/2, y+i*2+1, text, st)
			continue
		}
		lst := g.st(g.pal.dim)
		vst := g.st(g.pal.fg)
		if sel {
			lst = g.st(g.pal.accent)
			vst = g.st(g.pal.accent).Bold(true)
			switch row {
			case rowPlayer2, rowPort:
				value += "▏"
			default:
				value = "◀ " + value + " ▶"
			}
		}
		drawText(g.screen, lx, y+i*2, label, lst)
		drawText(g.screen, lx+24, y+i*2, value, vst)
	}
	if g.setupMode == modeAI {
		hints := map[string]string{
			diffEasy:   "Ratau drops dice anywhere.",
			diffMedium: "Flinky grabs the biggest swing each turn.",
			diffHard:   "Shrumy weighs every roll you could get back.",
		}
		drawCentered(g.screen, w/2, y+len(rows)*2+2, hints[g.save.Settings.Difficulty], g.st(g.pal.dim))
	}
	if g.note != "" {
		for i, l := range wrap(g.note, w-4) {
			drawCentered(g.screen, w/2, y+len(rows)*2+4+i, l, g.st(g.pal.lose))
		}
	}
	drawCentered(g.screen, w/2, h-1, "↑↓ select · ←→ change · Enter · Esc back", g.st(g.pal.dim))
}

func (g *game) drawPrompt(title, label, value string) {
	w, h := g.width, g.height
	g.drawTitle(1)
	bw := 50
	if bw > w-2 {
		bw = w - 2
	}
	bx, by := w/2-bw/2, h/2-4
	g.drawBox(bx, by, bw, 7, title)
	drawText(g.screen, bx+3, by+2, label, g.st(g.pal.dim))
	field := tcell.StyleDefault.Background(g.pal.emptyBg).Foreground(g.pal.fg)
	fillRect(g.screen, bx+3, by+4, bw-6, 1, field)
	shown := value
	for textWidth(shown) > bw-8 {
		shown = string([]rune(shown)[1:])
	}
	x := drawText(g.screen, bx+4, by+4, shown, field)
	g.screen.SetContent(x, by+4, '▏', nil, field.Foreground(g.pal.accent))
	if g.note != "" {
		drawCentered(g.screen, w/2, by+8, g.note, g.st(g.pal.lose))
	}
	drawCentered(g.screen, w/2, h-1, "Type · Enter confirm · Esc back", g.st(g.pal.dim))
}

func lobbyErrorText(e string) string {
	switch e {
	case errRoomFull:
		return "That game already has two players."
	case errOldProtocol:
		return "This game version is no longer compatible with the host."
	}
	return "Could not connect: " + e
}

func formatWait(d time.Duration) string {
	sec := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func (g *game) drawLobby() {
	w, h := g.width, g.height
	g.drawTitle(1)
	l, _ := g.ctl.(lobby)
	if l == nil {
		return
	}
	bw := 56
	if bw > w-2 {
		bw = w - 2
	}
	bx, by := w/2-bw/2, 4
	if port := l.hostPort(); port > 0 {
		addrs := localAddrs()
		bh := 10 + len(addrs)
		g.drawBox(bx, by, bw, bh, "Waiting for an opponent")
		drawText(g.screen, bx+3, by+2, "Share one of these addresses with your opponent:", g.st(g.pal.dim))
		row := by + 4
		for _, a := range addrs {
			drawCentered(g.screen, w/2, row, fmt.Sprintf("%s:%d", a, port), g.st(g.pal.accent).Bold(true))
			row++
		}
		drawCentered(g.screen, w/2, row, fmt.Sprintf("localhost:%d (this computer)", port), g.st(g.pal.dim))
		row += 2
		drawCentered(g.screen, w/2, row, "Waiting time: "+formatWait(time.Since(l.waitingSince())), g.st(g.pal.fg))
		st := g.save.Settings
		drawCentered(g.screen, w/2, row+1, roundsLabel(st.BestOf)+" · "+rulesSummary(st.Rules), g.st(g.pal.dim))
		drawCentered(g.screen, w/2, h-1, "Esc cancel", g.st(g.pal.dim))
		return
	}
	g.drawBox(bx, by, bw, 8, "Join an online game")
	if e := l.lobbyError(); e != "" {
		for i, line := range wrap(lobbyErrorText(e), bw-6) {
			drawCentered(g.screen, w/2, by+2+i, line, g.st(g.pal.lose))
		}
		drawCentered(g.screen, w/2, by+5, g.save.Settings.JoinAddr, g.st(g.pal.dim))
		drawCentered(g.screen, w/2, h-1, "Enter retry · Esc back", g.st(g.pal.dim))
		return
	}
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/400%4))
	drawCentered(g.screen, w/2, by+2, "Waiting for game to start"+dots, g.st(g.pal.fg))
	drawCentered(g.screen, w/2, by+4, g.save.Settings.JoinAddr, g.st(g.pal.accent))
	drawCentered(g.screen, w/2, h-1, "Esc cancel", g.st(g.pal.dim))
}

func rulesSummary(r Rules) string {
	var parts []string
	if r.NoDestroy {
		parts = append(parts, "no destruction")
	}
	if r.TwoDice {
		parts = append(parts, "two dice")
	}
	if r.FillBoth {
		parts = append(parts, "fill both boards")
	}
	if len(parts) == 0 {
		return "standard rules"
	}
	return strings.Join(parts, ", ")
}

// ---- play screen ----

type gridMarks struct {
	ghostCol, ghostSlot int // die about to be placed (preview)
	doomCol, doomVal    int // opponent dice the preview destroys
	lastCol, lastSlot   int // most recent placement
}

func noMarks() gridMarks { return gridMarks{-1, -1, -1, 0, -1, -1} }

type playLayout struct {
	top, gx, topGridY, botGridY, topScoreY, botScoreY, midY int
}

func (g *game) layout() playLayout {
	top := 0
	if g.height > 28 {
		top = (g.height - 28) / 2
	}
	l := playLayout{top: top, gx: (g.width - gridW) / 2}
	l.topScoreY = top + 2
	l.topGridY = top + 3
	l.midY = l.topGridY + gridH
	l.botGridY = l.midY + 1
	l.botScoreY = l.botGridY + gridH
	return l
}

func (l playLayout) cell(isTop bool, c, k int) (int, int) {
	x := l.gx + c*(dieW+colGap)
	if isTop {
		return x, l.topGridY + (gridRows-1-k)*(dieH+slotGap)
	}
	return x, l.botGridY + k*(dieH+slotGap)
}

func (g *game) drawGrid(l playLayout, grid *Grid, isTop bool, m gridMarks, ghostVal int) {
	for c := 0; c < gridCols; c++ {
		for k := 0; k < gridRows; k++ {
			x, y := l.cell(isTop, c, k)
			v := grid[c][k]
			switch {
			case c == m.ghostCol && k == m.ghostSlot:
				drawDie(g.screen, x, y, ghostVal, g.pal.ghostBg, g.pal.ghostPip)
			case v == 0:
				fillRect(g.screen, x, y, dieW, dieH, tcell.StyleDefault.Background(g.pal.emptyBg))
			case c == m.doomCol && v == m.doomVal:
				drawDie(g.screen, x, y, v, g.pal.doomBg, g.pal.doomPip)
			default:
				bg, pip := g.dieColors(grid.Matches(c, v))
				if c == m.lastCol && k == m.lastSlot {
					pip = g.pal.lastPip
				}
				drawDie(g.screen, x, y, v, bg, pip)
			}
		}
	}
}

func (g *game) drawColumnScores(l playLayout, y int, grid *Grid, preview *Grid, changed tcell.Color) {
	for c := 0; c < gridCols; c++ {
		x, _ := l.cell(true, c, 0)
		now := grid.ColumnScore(c)
		text, st := fmt.Sprint(now), g.st(g.pal.fg).Bold(true)
		if preview != nil {
			if next := preview.ColumnScore(c); next != now {
				text, st = fmt.Sprintf("%d→%d", now, next), g.st(changed).Bold(true)
			}
		}
		drawCentered(g.screen, x+dieW/2, y, text, st)
	}
}

func (g *game) drawPlay() {
	s := g.ctl.state()
	if s == nil {
		return
	}
	w, h := g.width, g.height
	l := g.layout()
	now := time.Now()

	bottom := g.ctl.seat()
	if bottom < 0 {
		bottom = 0
	}
	topP := 1 - bottom
	m := &s.Match

	// Preview for the player choosing a column.
	mover := m.Turn
	var preview *Match
	var pl Placement
	showPreview := s.Phase == phasePlay && g.myTurn(s) && !g.rolling() && g.overlay == overlayNone
	mv := Move{Col: g.cursor, Die: g.dieSel}
	if showPreview && m.Legal(mv) {
		p, next := m.Preview(mv)
		pl, preview = p, &next
	}

	for _, p := range []int{topP, bottom} {
		isTop := p == topP
		marks := noMarks()
		grid := &m.Grids[p]
		ghostVal := 0
		if s.Last != nil && s.Last.Player == p && s.Phase == phasePlay {
			marks.lastCol, marks.lastSlot = s.Last.Col, s.Last.Slot
			if marks.lastSlot >= 0 && grid[marks.lastCol][marks.lastSlot] != s.Last.Die {
				marks.lastCol = -1
			}
		}
		// The mover's grid is drawn as it would look after the placement so
		// new pairs and triples show their colors; the other grid marks the
		// dice that would be destroyed.
		var scorePreview *Grid
		var scoreColor tcell.Color
		switch {
		case preview != nil && p == mover:
			marks.ghostCol, marks.ghostSlot = pl.Col, pl.Slot
			ghostVal = pl.Die
			scorePreview, scoreColor = &preview.Grids[p], g.pal.win
			g.drawGrid(l, &preview.Grids[p], isTop, marks, ghostVal)
		case preview != nil:
			if !m.Rules.NoDestroy {
				marks.doomCol, marks.doomVal = pl.Col, pl.Die
			}
			scorePreview, scoreColor = &preview.Grids[p], g.pal.lose
			g.drawGrid(l, grid, isTop, marks, 0)
		default:
			g.drawGrid(l, grid, isTop, marks, 0)
		}
		y := l.botScoreY
		if isTop {
			y = l.topScoreY
		}
		g.drawColumnScores(l, y, grid, scorePreview, scoreColor)
	}

	// Column marker between the grids.
	if s.Phase == phasePlay && g.myTurn(s) && !g.rolling() {
		x, _ := l.cell(true, g.cursor, 0)
		arrow := "▼"
		if mover == topP {
			arrow = "▲"
		}
		mst := g.st(g.pal.accent).Bold(true)
		if !m.Grids[mover].Open(g.cursor) {
			mst = g.st(g.pal.dim)
		}
		drawCentered(g.screen, x+dieW/2, l.midY, arrow+" "+fmt.Sprint(g.cursor+1)+" "+arrow, mst)
	}

	g.drawHeader(s, l, now)
	g.drawPlayerPanel(s, l, topP, true, pl, preview != nil, now)
	g.drawPlayerPanel(s, l, bottom, false, pl, preview != nil, now)
	g.drawSidePanel(s, l)

	keys := "←→ column · 1-3/Enter place · L log · H history · R resign · Esc leave"
	if m.Rules.TwoDice {
		keys = "←→ column · Tab die · Enter place · L log · H history · R resign · Esc"
	}
	if s.Phase != phasePlay {
		keys = "←→ choose · Enter · V view board · L log · H history · Esc leave"
	}
	drawCentered(g.screen, w/2, h-1, keys, g.st(g.pal.dim))

	switch {
	case g.overlay == overlayLog:
		g.drawLogOverlay(s)
	case g.overlay == overlayHistory:
		g.drawHistoryOverlay(s)
	case g.overlay == overlayResign:
		warn := "Resigning ends the game immediately and counts as a loss."
		if s.Mode == modeLocal {
			warn = s.Names[m.Turn] + " gives the game to " + s.Names[1-m.Turn] + "."
		}
		g.drawConfirm("Resign this match?", warn, "Cancel", "Confirm resignation")
	case g.overlay == overlayLeave:
		warn := "This game is not saved."
		if s.Mode == modeOnline {
			warn = "Leaving counts as resigning."
		}
		g.drawConfirm("Leave this game?", warn, "Stay", "Leave")
	case s.Phase != phasePlay && !g.hideResult:
		g.drawResult(s, l)
	}
}

func (g *game) drawHeader(s *session, l playLayout, now time.Time) {
	w := g.width
	y := l.top
	var text string
	var st tcell.Style
	if p, left := g.ctl.disconnectLeft(now); p >= 0 {
		secs := int(left.Seconds() + 0.999)
		if g.isYou(s, 1-p) {
			text = fmt.Sprintf("%s disconnected. They have %d seconds to reconnect.", s.Names[p], secs)
		} else {
			text = fmt.Sprintf("Connection lost. Reconnecting… %ds", secs)
		}
		st = g.st(g.pal.lose).Bold(true)
	} else if s.Phase == phasePlay {
		p := s.Match.Turn
		if g.isYou(s, p) {
			text = "Your turn"
		} else {
			text = s.Names[p] + "'s turn"
		}
		st = g.st(g.pal.accent).Bold(true)
	} else {
		text = "Round over"
		if s.Phase == phaseGameOver {
			text = "Game over"
		}
		st = g.st(g.pal.fg).Bold(true)
	}
	drawText(g.screen, 1, y, "KNUCKLEBONES", g.st(g.pal.title).Bold(true))
	drawCentered(g.screen, w/2, y, text, st)
	label := fmt.Sprintf("Round %d · %s", s.Round, s.roundsLabel())
	drawText(g.screen, w-1-textWidth(label), y, label, g.st(g.pal.dim))
}

func (g *game) drawPlayerPanel(s *session, l playLayout, p int, isTop bool, pl Placement, hasPreview bool, now time.Time) {
	pw := l.gx - 3
	if pw < 10 {
		return
	}
	x := 1
	m := &s.Match
	turn := s.Phase == phasePlay && m.Turn == p

	var nameY, trayY int
	if isTop {
		nameY, trayY = l.topGridY, l.topGridY+gridH-dieH-1
	} else {
		trayY, nameY = l.botGridY+1, l.botGridY+gridH-4
	}

	nameSt := g.st(g.pal.fg).Bold(true)
	if turn {
		nameSt = g.st(g.pal.accent).Bold(true)
	}
	name := s.Names[p]
	if g.isYou(s, p) {
		name += " (you)"
	}
	if !s.Connected[p] {
		name += " ⚠"
	}
	drawText(g.screen, x, nameY, truncate(name, pw), nameSt)
	drawText(g.screen, x, nameY+1, "Total ", g.st(g.pal.dim))
	drawText(g.screen, x+6, nameY+1, fmt.Sprint(m.Grids[p].Score()), g.st(g.pal.fg).Bold(true))

	info := ""
	if !s.freePlay() && s.BestOf > 1 {
		need := s.BestOf/2 + 1
		info = "Wins " + strings.Repeat("●", s.Wins[p]) + strings.Repeat("○", max(0, need-s.Wins[p]))
	} else if s.Round > 1 || len(s.History) > 0 {
		info = fmt.Sprintf("Wins %d", s.Wins[p])
	}
	if info != "" {
		drawText(g.screen, x, nameY+2, info, g.st(g.pal.dim))
	}
	if s.TurnLimit > 0 && s.Timeouts[p] > 0 {
		drawText(g.screen, x, nameY+3, fmt.Sprintf("Timeouts: %d / %d", s.Timeouts[p], maxTimeouts), g.st(g.pal.lose))
	}

	if !turn {
		return
	}
	rolling := g.rolling()
	for i := 0; i < 2; i++ {
		v := m.Dice[i]
		if v == 0 {
			continue
		}
		dx := x + i*(dieW+2)
		if rolling {
			drawDie(g.screen, dx, trayY, rollingFace(i), g.pal.dieBg, g.pal.pip)
			continue
		}
		bg, pip := g.pal.dieBg, g.pal.pip
		if m.Dice[1] != 0 && i != g.dieSel && g.myTurn(s) {
			bg, pip = g.pal.ghostBg, g.pal.ghostPip
		}
		drawDie(g.screen, dx, trayY, v, bg, pip)
	}
	ty := trayY + dieH
	if isTop {
		ty = trayY - 1
	}
	switch {
	case hasPreview:
		t := fmt.Sprintf("+%d", pl.Gain)
		if pl.Destroyed > 0 {
			t += fmt.Sprintf(" · destroys %d (−%d)", pl.Destroyed, pl.Lost)
		}
		drawText(g.screen, x, ty, truncate(t, pw), g.st(g.pal.accent))
	case !g.myTurn(s) && !rolling:
		drawText(g.screen, x, ty, "Thinking…", g.st(g.pal.dim))
	}
	if left := g.ctl.turnLeft(now); left > 0 {
		ly := ty + 1
		if isTop {
			ly = ty - 1
		}
		secs := int(left.Seconds() + 0.999)
		c := g.pal.fg
		if secs <= 5 {
			c = g.pal.lose
		}
		drawText(g.screen, x, ly, fmt.Sprintf("Time left: %ds", secs), g.st(c))
	}
}

func truncate(s string, w int) string {
	if textWidth(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func (g *game) drawSidePanel(s *session, l playLayout) {
	x := l.gx + gridW + 3
	pw := g.width - x - 1
	if pw < 14 {
		return
	}
	y := l.topGridY
	if s.BestOf != 1 || s.FreePlay || len(s.History) > 0 {
		drawText(g.screen, x, y, truncate(fmt.Sprintf("Series %d – %d", s.Wins[0], s.Wins[1]), pw), g.st(g.pal.fg).Bold(true))
		drawText(g.screen, x, y+1, truncate(s.Names[0]+" v "+s.Names[1], pw), g.st(g.pal.dim))
		y += 3
	}
	if s.Rules != (Rules{}) {
		for _, line := range wrap(rulesSummary(s.Rules), pw) {
			drawText(g.screen, x, y, line, g.st(g.pal.dim))
			y++
		}
		y++
	}
	drawText(g.screen, x, y, "Log", g.st(g.pal.accent).Bold(true))
	y++
	// Newest entries first, drawn top-down; the newest entry is brighter.
	room := l.botScoreY - y + 1
	for i := len(s.Log) - 1; i >= 0 && room > 0; i-- {
		c := g.pal.dim
		if i == len(s.Log)-1 {
			c = g.pal.fg
		}
		wrapped := wrap(g.logText(s, s.Log[i]), pw-2)
		for j, line := range wrapped {
			if room == 0 {
				break
			}
			prefix := "  "
			if j == 0 {
				prefix = "· "
			}
			drawText(g.screen, x, y, prefix+line, g.st(c))
			y++
			room--
		}
	}
}

func (g *game) drawResult(s *session, l playLayout) {
	w := g.width
	bw := 58
	if bw > w-2 {
		bw = w - 2
	}
	msg, c := g.resultText(s)
	lines := wrap(msg, bw-6)
	bh := 9 + len(lines)
	bx, by := w/2-bw/2, l.midY-bh/2
	if by < 1 {
		by = 1
	}
	title := "Round over"
	if s.Phase == phaseGameOver {
		title = "Game over"
	}
	g.drawOverlayBox(bx, by, bw, bh, title)
	y := by + 2
	for _, line := range lines {
		drawCentered(g.screen, w/2, y, line, g.st(c).Bold(true))
		y++
	}
	sc := s.Match.Scores()
	drawCentered(g.screen, w/2, y, fmt.Sprintf("%s %d  –  %d %s", s.Names[0], sc[0], sc[1], s.Names[1]), g.st(g.pal.fg))
	y++
	if s.BestOf != 1 || s.FreePlay {
		drawCentered(g.screen, w/2, y, fmt.Sprintf("Rounds won %d – %d · %s", s.Wins[0], s.Wins[1], s.roundsLabel()), g.st(g.pal.dim))
	}
	y++
	status := g.recordNote
	stc := g.pal.accent
	if s.Mode == modeOnline {
		me := g.ctl.seat()
		opp := 1 - me
		switch {
		case g.ctl.opponentGone():
			status, stc = "Your opponent has left the room.", g.pal.lose
		case s.Votes[me] != "" && s.Votes[opp] != s.Votes[me]:
			status, stc = "Waiting for "+s.Names[opp]+"...", g.pal.dim
		case s.Votes[opp] == voteRematch:
			status = s.Names[opp] + " wants to rematch!"
		case s.Votes[opp] == voteContinue:
			status = s.Names[opp] + " wants to continue!"
		case s.Votes[opp] == voteFree:
			status = s.Names[opp] + " wants to go into free play!"
		}
	}
	if status != "" {
		drawCentered(g.screen, w/2, y, status, g.st(stc))
	}
	y += 2
	choices := g.resultChoices(s)
	labels := make([]string, len(choices))
	total := 0
	for i, ch := range choices {
		labels[i] = " " + choiceLabel(ch) + " "
		total += textWidth(labels[i]) + 2
	}
	x := w/2 - total/2
	for i, lb := range labels {
		st := g.st(g.pal.fg)
		if i == g.resultIndex {
			st = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
		}
		x = drawText(g.screen, x, y, lb, st) + 2
	}
}

func (g *game) drawConfirm(title, warn, no, yes string) {
	w, h := g.width, g.height
	bw := 56
	if bw > w-2 {
		bw = w - 2
	}
	lines := wrap(warn, bw-6)
	bh := 7 + len(lines)
	bx, by := w/2-bw/2, h/2-bh/2
	g.drawOverlayBox(bx, by, bw, bh, title)
	for i, line := range lines {
		drawCentered(g.screen, w/2, by+2+i, line, g.st(g.pal.fg))
	}
	y := by + bh - 3
	opts := []string{" " + no + " ", " " + yes + " "}
	total := textWidth(opts[0]) + textWidth(opts[1]) + 4
	x := w/2 - total/2
	for i, o := range opts {
		st := g.st(g.pal.fg)
		if i == g.confirmIndex {
			bg := g.pal.accent
			if i == 1 {
				bg = g.pal.lose
			}
			st = tcell.StyleDefault.Background(bg).Foreground(g.pal.bg).Bold(true)
		}
		x = drawText(g.screen, x, y, o, st) + 4
	}
}

func (g *game) drawScrollBox(title string, lines []string, colors []tcell.Color, fromBottom bool, hint string) {
	w, h := g.width, g.height
	bw := w - 8
	if bw > 72 {
		bw = 72
	}
	bh := h - 4
	bx, by := w/2-bw/2, 2
	g.drawOverlayBox(bx, by, bw, bh, title)
	room := bh - 4
	maxScroll := len(lines) - room
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.overlayScroll > maxScroll {
		g.overlayScroll = maxScroll
	}
	start := g.overlayScroll
	if fromBottom {
		start = maxScroll - g.overlayScroll
	}
	for i := 0; i < room && start+i < len(lines); i++ {
		drawText(g.screen, bx+3, by+2+i, lines[start+i], g.st(colors[start+i]))
	}
	drawCentered(g.screen, w/2, by+bh-1, " "+hint+" ", g.st(g.pal.dim))
}

func (g *game) drawLogOverlay(s *session) {
	bw := g.width - 8
	if bw > 72 {
		bw = 72
	}
	var lines []string
	var colors []tcell.Color
	for _, e := range s.Log {
		c := g.pal.fg
		switch e.Kind {
		case logFirst:
			c = g.pal.accent
		case logRound, logForfeit:
			c = g.pal.win
		case logTimeout:
			c = g.pal.lose
		}
		for _, line := range wrap(g.logText(s, e), bw-6) {
			lines = append(lines, line)
			colors = append(colors, c)
		}
	}
	if len(lines) == 0 {
		lines, colors = []string{"Nothing yet."}, []tcell.Color{g.pal.dim}
	}
	g.drawScrollBox("Logs", lines, colors, true, "↑↓ scroll · L or Esc close")
}

func (g *game) drawHistoryOverlay(s *session) {
	var lines []string
	var colors []tcell.Color
	add := func(t string, c tcell.Color) {
		lines = append(lines, t)
		colors = append(colors, c)
	}
	add(fmt.Sprintf("Round %d of %s", s.Round, s.roundsLabel()), g.pal.accent)
	add("", g.pal.fg)
	if len(s.History) == 0 {
		add("The first round isn't finished yet!", g.pal.dim)
	}
	for _, e := range s.History {
		add(fmt.Sprintf("Round %d   %s %d – %d %s", e.Round, s.Names[0], e.Scores[0], e.Scores[1], s.Names[1]), g.pal.fg)
		switch {
		case e.Winner < 0:
			add("    This is a tie!", g.pal.dim)
		case g.isYou(s, e.Winner):
			add("    You won!", g.pal.win)
		default:
			add("    "+s.Names[e.Winner]+" won!", g.pal.dim)
		}
		switch {
		case e.Lead < 0:
		case g.isYou(s, e.Lead):
			add("    You are taking the lead!", g.pal.accent)
		default:
			add("    "+s.Names[e.Lead]+" is taking the lead!", g.pal.accent)
		}
	}
	if s.Ties > 0 {
		add("", g.pal.fg)
		add(fmt.Sprintf("Ties: %d", s.Ties), g.pal.dim)
	}
	g.drawScrollBox("History", lines, colors, false, "↑↓ scroll · H or Esc close")
}

// ---- guide and stats ----

type guideLine struct {
	text string
	kind int // 0 body, 1 heading, 2 table
}

var guideText = []guideLine{
	{"How to play", 1},
	{"Each turn, you get a random die to place in a column. The goal is to score more points than your opponent when the game ends, which happens when one of the boards is full.", 0},
	{"• You score even more points when stacking the same dice in a column.", 0},
	{"• You remove all of your opponent's dice with the same value when placing a die in the matching column.", 0},
	{"Taking a turn", 1},
	{"1. A die is rolled for you. A coin toss decides who starts each round.", 0},
	{"2. Place it in any of your columns with an empty slot. Columns fill from the middle of the table outward, and placed dice never move by choice.", 0},
	{"3. Every die of the same value in your opponent's facing column is destroyed. Their other columns are untouched.", 0},
	{"Column scoring", 1},
	{"A die is worth its face value times the number of dice of that value in its column. For each value v appearing n times, the column scores n × n × v.", 0},
	{"  Column      Calculation           Score", 2},
	{"  6           6                         6", 2},
	{"  6 2         6 + 2                     8", 2},
	{"  6 6         (6×2) + (6×2)            24", 2},
	{"  6 6 2       24 + 2                   26", 2},
	{"  4 1 4       (4×2) + 1 + (4×2)        17", 2},
	{"  6 6 6       (6×3) × 3                54", 2},
	{"  3 3 3       (3×3) × 3                27", 2},
	{"  1 1 1       (1×3) × 3                 9", 2},
	{"The best column is three 6s for 54, so the best board is 162.", 0},
	{"Ending the game", 1},
	{"The game stops the moment either board is full, even mid-comeback. The higher total wins; equal totals are a tie. A best-of series continues until someone wins a majority of rounds. Free play keeps going round after round.", 0},
	{"Reading the board", 1},
	{"Ivory dice are singles, gold dice are pairs, blue dice are triples. While you choose a column, a shaded die shows where yours will land, red dice are the ones it would destroy, and column totals show the change. Red pips mark the last die placed.", 0},
	{"House rules", 1},
	{"Set these in Game settings before you start.", 0},
	{"• Destroy matching dice: Off plays the quiet, stacking-only game.", 0},
	{"• Dice per turn: Two rolls two dice and you place one. Tab switches between them.", 0},
	{"• Game ends when: Both boards are full removes the close-the-board tactic.", 0},
	{"Strategy", 1},
	{"• Value swings, not scores. Gaining 6 while taking 24 away is worth 30.", 0},
	{"• Stack high, spread low. Two 6s are 24; two 1s are only 4.", 0},
	{"• Stacks get safer near the end. Your opponent rolls the value that destroys a stack 1 time in 6, every turn.", 0},
	{"• Answer a stack with a single die: a lone 5 across from their 5, 5 removes 20.", 0},
	{"• Control the clock. Ahead? Fill your board. Behind? Keep a column open so you are not forced to end the game.", 0},
	{"• Use quiet columns, where the opposite dice cannot match yours, for low dice.", 0},
	{"Opponents", 1},
	{"• Ratau (Easy) drops dice in any open column.", 0},
	{"• Flinky (Medium) takes the biggest immediate swing.", 0},
	{"• Shrumy (Hard) averages your best reply over every roll, looks one step further, and knows when ending the game wins or loses it.", 0},
	{"Online play", 1},
	{"One player picks Host an online game and shares an address shown in the lobby; the other picks Join and types it. The host rolls the dice. Turns can be timed; three timeouts lose the match. A player who drops has 30 seconds to reconnect. After a game, both players choose the same option to play again.", 0},
	{"Credits", 1},
	{"The original Knucklebones game in Cult of the Lamb was created by Massive Monster. This is a fan-made game and not an official implementation. Rules and strategy notes follow the dice.now Knucklebones guide.", 0},
}

func (g *game) drawGuide() {
	w, h := g.width, g.height
	tw := w - 6
	if tw > 74 {
		tw = 74
	}
	x := w/2 - tw/2
	var lines []string
	var styles []tcell.Style
	for i, gl := range guideText {
		switch gl.kind {
		case 1:
			if i > 0 {
				lines, styles = append(lines, ""), append(styles, g.st(g.pal.fg))
			}
			lines, styles = append(lines, gl.text), append(styles, g.st(g.pal.accent).Bold(true))
		case 2:
			lines, styles = append(lines, gl.text), append(styles, g.st(g.pal.fg))
		default:
			for j, l := range wrap(gl.text, tw-2) {
				if j > 0 && (strings.HasPrefix(gl.text, "•") || (len(gl.text) > 1 && gl.text[1] == '.')) {
					l = "  " + l
				}
				lines, styles = append(lines, l), append(styles, g.st(g.pal.fg))
			}
		}
	}
	g.drawTitle(0)
	room := h - 3
	maxScroll := len(lines) - room
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.guideScroll > maxScroll {
		g.guideScroll = maxScroll
	}
	for i := 0; i < room && g.guideScroll+i < len(lines); i++ {
		drawText(g.screen, x, 2+i, lines[g.guideScroll+i], styles[g.guideScroll+i])
	}
	hint := "↑↓ scroll · PgUp/PgDn · Esc back"
	if maxScroll > 0 {
		hint = fmt.Sprintf("%s · %d%%", hint, g.guideScroll*100/maxScroll)
	}
	drawCentered(g.screen, w/2, h-1, hint, g.st(g.pal.dim))
}

func (g *game) drawStats() {
	w, h := g.width, g.height
	g.drawTitle(1)
	drawCentered(g.screen, w/2, 3, "Stats", g.st(g.pal.fg).Bold(true))
	tw := 70
	if tw > w-2 {
		tw = w - 2
	}
	x := w/2 - tw/2
	y := 5
	header := fmt.Sprintf("%-22s %6s %5s %7s %6s %5s %7s", "", "Games", "Wins", "Losses", "Draws", "Best", "Series")
	drawText(g.screen, x, y, header, g.st(g.pal.dim))
	y++
	labels := map[string]string{
		"ai-" + diffEasy:   "vs Ratau (Easy)",
		"ai-" + diffMedium: "vs Flinky (Medium)",
		"ai-" + diffHard:   "vs Shrumy (Hard)",
		recordOnline:       "Online",
		recordLocal:        "Same keyboard",
	}
	for _, k := range recordKeys {
		r := g.save.Records[k]
		if r == nil {
			r = &record{}
		}
		var line string
		if k == recordLocal {
			line = fmt.Sprintf("%-22s %6d %5s %7s %6d %5d %7s", labels[k], r.Games, "–", "–", r.Draws, r.Best, "–")
		} else {
			series := "–"
			if r.SeriesWins+r.SeriesLosses > 0 {
				series = fmt.Sprintf("%d–%d", r.SeriesWins, r.SeriesLosses)
			}
			line = fmt.Sprintf("%-22s %6d %5d %7d %6d %5d %7s", labels[k], r.Games, r.Wins, r.Losses, r.Draws, r.Best, series)
		}
		c := g.pal.fg
		if r.Games == 0 {
			c = g.pal.dim
		}
		drawText(g.screen, x, y, line, g.st(c))
		y++
	}
	y += 1
	drawText(g.screen, x, y, "Top 10 final scores", g.st(g.pal.accent).Bold(true))
	y++
	if len(g.save.HighScores) == 0 {
		drawText(g.screen, x, y, "Finish a game and your score is added here.", g.st(g.pal.dim))
	}
	for i, e := range g.save.HighScores {
		if y >= h-2 {
			break
		}
		line := fmt.Sprintf("%2d. %-16s %4d   vs %-16s %4d   %s", i+1, e.Name, e.Score, e.Opponent, e.Against, e.Date)
		drawText(g.screen, x, y, truncate(line, tw), g.st(g.pal.fg))
		y++
	}
	drawCentered(g.screen, w/2, h-1, "Esc back", g.st(g.pal.dim))
}
