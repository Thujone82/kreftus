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
	case viewNetwork:
		g.drawNetwork()
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
	drawDieMarked(s, x, y, v, markNone, bg, pip)
}

func drawDieMarked(s tcell.Screen, x, y, v, mark int, bg, pip tcell.Color) {
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
	if r := tuiMarkRune(mark); r != 0 {
		s.SetContent(x+dieW-1, y, r, nil, st.Bold(true))
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

	labels := g.menuLabels()
	y := top + 9
	for i, l := range labels {
		st := g.st(g.pal.fg)
		if i == g.menuIndex {
			st = g.st(g.pal.accent).Bold(true)
			l = "▶ " + l + " ◀"
		}
		drawCentered(g.screen, w/2, y+i, l, st)
	}
	drawCentered(g.screen, w/2, h-4, creditLine1, g.st(g.pal.dim))
	drawCentered(g.screen, w/2, h-3, creditLine2, g.st(g.pal.dim))
	drawCentered(g.screen, w/2, h-1, "↑↓ select · Enter · T theme · Esc quit", g.st(g.pal.dim))
}

func (g *game) drawSettings() {
	w, h := g.width, g.height
	g.drawTitle(1)
	drawCentered(g.screen, w/2, 3, setupTitles[g.setupMode], g.st(g.pal.fg).Bold(true))
	sub := "Game settings"
	if g.setupMode == setupOptions {
		sub = "Appearance & audio"
	}
	drawCentered(g.screen, w/2, 4, sub, g.st(g.pal.dim))

	rows := g.settingRows()
	step := 2
	y := 6
	if y+len(rows)*2+3 > h-1 {
		step = 1
		y = 5
	}
	lx := w/2 - 24
	if lx < 1 {
		lx = 1
	}
	for i, row := range rows {
		label, value := g.settingText(row)
		sel := i == g.setIndex
		if row == rowStart || row == rowDone {
			st := g.st(g.pal.fg)
			btn := "  " + label + "  "
			if sel {
				st = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
				btn = "▶ " + label + " ◀"
			}
			ry := y + i*step + 1
			if ry < h-1 {
				drawCentered(g.screen, w/2, ry, btn, st)
			}
			continue
		}
		lst := g.st(g.pal.dim)
		vst := g.st(g.pal.fg)
		if sel {
			lst = g.st(g.pal.accent)
			vst = g.st(g.pal.accent).Bold(true)
			switch row {
			case rowPlayer2:
				value = withCaret(value, g.inputPos)
			case rowPort:
				value = withCaret(g.portEditString(), g.inputPos)
			case rowRoomCode:
				value = withCaret(g.hostRoomCode, g.inputPos)
			default:
				value = "◀ " + value + " ▶"
			}
		} else if row == rowRoomCode && g.hostRoomCode == "" {
			vst = g.st(g.pal.dim)
		}
		drawText(g.screen, lx, y+i*step, label, lst)
		drawText(g.screen, lx+22, y+i*step, value, vst)
	}
	hintY := y + len(rows)*step + 1
	if step == 2 {
		hintY++
	}
	if hintY < h-1 {
		switch g.setupMode {
		case modeAI:
			drawCentered(g.screen, w/2, hintY, difficultyHints[g.save.Settings.Difficulty], g.st(g.pal.dim))
		case setupHost, setupOnlineHost:
			if g.setIndex >= 0 && g.setIndex < len(rows) && rows[g.setIndex] == rowRoomCode {
				drawCentered(g.screen, w/2, hintY, "Optional room code locks joins & enables wire MACs.", g.st(g.pal.dim))
			}
		}
	}
	if g.note != "" {
		ny := hintY + 1
		if step == 2 {
			ny++
		}
		for i, l := range wrap(g.note, w-4) {
			if ny+i < h-1 {
				drawCentered(g.screen, w/2, ny+i, l, g.st(g.pal.lose))
			}
		}
	}
	drawCentered(g.screen, w/2, h-1, "↑↓ select · ←→ change · Enter · Esc back", g.st(g.pal.dim))
}

func withCaret(s string, pos int) string {
	r := []rune(s)
	pos = clampRunePos(s, pos)
	return string(r[:pos]) + "▏" + string(r[pos:])
}

func drawFieldSlice(s tcell.Screen, x, y, fieldW int, value string, pos int, active bool, field, dim tcell.Style, placeholder string) {
	fillRect(s, x, y, fieldW, 1, field)
	if !active && value == "" && placeholder != "" {
		drawText(s, x+1, y, placeholder, dim)
		return
	}
	r := []rune(value)
	pos = clampRunePos(value, pos)
	start := 0
	maxW := fieldW - 2
	for textWidth(string(r[start:pos])+"▏") > maxW && start < pos {
		start++
	}
	end := len(r)
	for textWidth(string(r[start:end])+"▏") > maxW && end > pos {
		end--
	}
	shown := string(r[start:end])
	if active {
		shown = withCaret(shown, pos-start)
	}
	drawText(s, x+1, y, shown, field)
}

func formatCompactUint(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatRoomMeta(rm lobbyRoomInfo) string {
	parts := []string{roundsLabel(rm.BestOf)}
	if summary := rulesSummary(rm.Rules); summary != "standard rules" {
		parts = append(parts, summary)
	}
	if rm.TurnSeconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", rm.TurnSeconds))
	}
	return strings.Join(parts, " · ")
}

func (g *game) drawNetwork() {
	w, h := g.width, g.height
	g.drawTitle(1)

	bw := 68
	if bw > w-2 {
		bw = w - 2
	}
	bh := h - 6
	if bh < 14 {
		bh = 14
	}
	if bh > 20 {
		bh = 20
	}
	bx, by := w/2-bw/2, 3
	if by+bh >= h-1 {
		by = 2
	}
	g.drawBox(bx, by, bw, bh, "Network Play")

	// Mode toggle row: [ Online ]  [ Direct ]
	onLbl := "  Online (vps.kreft.us)  "
	dirLbl := "  Direct  "
	onSt, dirSt := g.st(g.pal.dim), g.st(g.pal.dim)
	if g.netOnline {
		onLbl = "▶ Online (vps.kreft.us) ◀"
		if g.netFocus == netFocusMode {
			onSt = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
		} else {
			onSt = g.st(g.pal.accent).Bold(true)
		}
	} else {
		dirLbl = "▶ Direct ◀"
		if g.netFocus == netFocusMode {
			dirSt = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
		} else {
			dirSt = g.st(g.pal.accent).Bold(true)
		}
	}
	toggleW := textWidth(onLbl) + 2 + textWidth(dirLbl)
	tx := w/2 - toggleW/2
	tx = drawText(g.screen, tx, by+1, onLbl, onSt) + 2
	drawText(g.screen, tx, by+1, dirLbl, dirSt)

	field := tcell.StyleDefault.Background(g.pal.emptyBg).Foreground(g.pal.fg)
	dimField := tcell.StyleDefault.Background(g.pal.emptyBg).Foreground(g.pal.dim)
	accentBtn := tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
	idleBtn := tcell.StyleDefault.Background(g.pal.emptyBg).Foreground(g.pal.fg)

	if g.netOnline {
		snap := g.networkSnapshot()
		statsLine := fmt.Sprintf("In lobby: %d · Live: %d · Completed: %s · Dice: %s",
			snap.Stats.LobbyWatchers, snap.Stats.LiveMatches, formatCompactUint(snap.Stats.CompletedMatches), formatCompactUint(snap.Stats.TotalDicePlayed))
		if !snap.Connected {
			if snap.Err != "" {
				statsLine = "Offline (" + truncate(snap.Err, bw-18) + ")"
			} else {
				statsLine = "Connecting to " + g.rendezvousAddr() + "..."
			}
		}
		drawCentered(g.screen, w/2, by+3, truncate(statsLine, bw-4), g.st(g.pal.dim))

		listTop := by + 5
		listH := bh - 11
		if listH < 3 {
			listH = 3
		}
		roomsTitle := fmt.Sprintf("Open Matches (%d)", len(snap.Rooms))
		titleSt := g.st(g.pal.dim)
		if g.netFocus == netFocusRooms {
			titleSt = g.st(g.pal.accent).Bold(true)
		}
		drawText(g.screen, bx+3, listTop-1, roomsTitle, titleSt)

		if len(snap.Rooms) == 0 {
			msg := "No unstarted matches waiting right now. Press H to host one!"
			if !snap.Connected {
				if snap.Err != "" {
					msg = "Unable to reach lobby (Press R to retry or switch to Direct)"
				} else {
					msg = "Connecting to rendezvous lobby..."
				}
			}
			drawCentered(g.screen, w/2, listTop+listH/2, truncate(msg, bw-6), g.st(g.pal.dim))
		} else {
			g.clampNetworkRoom(len(snap.Rooms))
			if g.netRoomIdx >= g.netRoomScroll+listH {
				g.netRoomScroll = g.netRoomIdx - listH + 1
			}
			for row := 0; row < listH && g.netRoomScroll+row < len(snap.Rooms); row++ {
				idx := g.netRoomScroll + row
				rm := snap.Rooms[idx]
				selected := idx == g.netRoomIdx
				rowSt := g.st(g.pal.fg)
				prefix := "  "
				if selected {
					prefix = "▸ "
					if g.netFocus == netFocusRooms {
						rowSt = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
					} else {
						rowSt = g.st(g.pal.accent).Bold(true)
					}
					fillRect(g.screen, bx+3, listTop+row, bw-6, 1, tcell.StyleDefault.Background(g.pal.emptyBg))
				}
				lockMark := ""
				if rm.Locked {
					lockMark = "🔒 "
				}
				waitStr := ""
				if rm.CreatedAt > 0 {
					waitStr = " · " + formatWait(time.Since(time.Unix(rm.CreatedAt, 0)))
				}
				line := fmt.Sprintf("%s%s%s · %s%s", prefix, lockMark, rm.Host, formatRoomMeta(rm), waitStr)
				drawText(g.screen, bx+3, listTop+row, truncate(line, bw-6), rowSt)
			}
		}

		codeY := listTop + listH + 1
		codeLblSt := g.st(g.pal.dim)
		if g.netFocus == netFocusCode {
			codeLblSt = g.st(g.pal.accent).Bold(true)
		}
		drawText(g.screen, bx+3, codeY, "Room code (for 🔒 locked match):", codeLblSt)
		drawFieldSlice(g.screen, bx+3, codeY+1, bw-6, g.joinCode, g.inputPos, g.netFocus == netFocusCode, field, dimField, "(optional)")

		btnY := by + bh - 2
		joinLbl := "  Join Selected  "
		hostLbl := "  Host Online Match  "
		joinSt, hostSt := idleBtn, idleBtn
		if g.netFocus == netFocusJoin {
			joinLbl = "▶ Join Selected ◀"
			joinSt = accentBtn
		} else if g.netFocus == netFocusHost {
			hostLbl = "▶ Host Online Match ◀"
			hostSt = accentBtn
		}
		btot := textWidth(joinLbl) + 3 + textWidth(hostLbl)
		rx := w/2 - btot/2
		rx = drawText(g.screen, rx, btnY, joinLbl, joinSt) + 3
		drawText(g.screen, rx, btnY, hostLbl, hostSt)

		if g.note != "" && by+bh < h-1 {
			drawCentered(g.screen, w/2, by+bh, truncate(g.note, w-4), g.st(g.pal.lose))
		}
		drawCentered(g.screen, w/2, h-1, truncate("↑↓←→ navigate · Tab · H host · M mode · Enter confirm · Esc back", w-2), g.st(g.pal.dim))
		return
	}

	// Direct Mode
	hostLbl := "  Host Direct Game  "
	hostSt := idleBtn
	if g.netFocus == netFocusHost {
		hostLbl = "▶ Host Direct Game ◀"
		hostSt = accentBtn
	}
	drawCentered(g.screen, w/2, by+3, hostLbl, hostSt)
	drawCentered(g.screen, w/2, by+4, "Open a port directly or enter a host address to join:", g.st(g.pal.dim))

	lblAddr, lblCode := g.st(g.pal.dim), g.st(g.pal.dim)
	if g.netFocus == netFocusAddr {
		lblAddr = g.st(g.pal.accent).Bold(true)
	} else if g.netFocus == netFocusCode {
		lblCode = g.st(g.pal.accent).Bold(true)
	}
	drawText(g.screen, bx+3, by+6, "Host address (host:port)", lblAddr)
	drawFieldSlice(g.screen, bx+3, by+7, bw-6, g.input, g.inputPos, g.netFocus == netFocusAddr, field, dimField, "")
	drawText(g.screen, bx+3, by+9, "Room code (optional)", lblCode)
	drawFieldSlice(g.screen, bx+3, by+10, bw-6, g.joinCode, g.inputPos, g.netFocus == netFocusCode, field, dimField, "(optional)")

	joinLbl := "  Join Direct  "
	joinSt := idleBtn
	if g.netFocus == netFocusJoin {
		joinLbl = "▶ Join Direct ◀"
		joinSt = accentBtn
	}
	drawCentered(g.screen, w/2, by+12, joinLbl, joinSt)
	if g.note != "" && by+bh < h-1 {
		drawCentered(g.screen, w/2, by+bh, truncate(g.note, w-4), g.st(g.pal.lose))
	}
	drawCentered(g.screen, w/2, h-1, truncate("↑↓←→ navigate · Tab · M mode · Enter join · Esc back", w-2), g.st(g.pal.dim))
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
	dimField := tcell.StyleDefault.Background(g.pal.emptyBg).Foreground(g.pal.dim)
	drawFieldSlice(g.screen, bx+3, by+4, bw-6, value, g.inputPos, true, field, dimField, "")
	if g.note != "" {
		drawCentered(g.screen, w/2, by+8, g.note, g.st(g.pal.lose))
	}
	drawCentered(g.screen, w/2, h-1, "Type · ←→ move · Enter confirm · Esc back", g.st(g.pal.dim))
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
		st := g.save.Settings
		addrs := lobbyHostAddresses(st.BindIP, port)
		rc := l.hostRoomCode()
		extra := 0
		if rc != "" {
			extra += 2
		}
		bh := 10 + len(addrs) + extra
		if bh > h-by-2 {
			bh = h - by - 2
		}
		g.drawBox(bx, by, bw, bh, "Waiting for an opponent")
		drawText(g.screen, bx+3, by+2, "Share an address with your opponent:", g.st(g.pal.dim))
		row := by + 4
		for _, a := range addrs {
			if row >= by+bh-4 {
				break
			}
			drawCentered(g.screen, w/2, row, a, g.st(g.pal.accent).Bold(true))
			row++
		}
		if st.BindIP != "127.0.0.1" && row < by+bh-3 {
			drawCentered(g.screen, w/2, row, fmt.Sprintf("localhost:%d (this computer)", port), g.st(g.pal.dim))
			row++
		}
		if rc != "" && row+1 < by+bh-2 {
			row++
			drawCentered(g.screen, w/2, row, "Room code: "+rc, g.st(g.pal.accent).Bold(true))
		}
		row += 2
		if row < by+bh-1 {
			drawCentered(g.screen, w/2, row, "Waiting time: "+formatWait(time.Since(l.waitingSince())), g.st(g.pal.fg))
		}
		if row+1 < by+bh-1 {
			drawCentered(g.screen, w/2, row+1, roundsLabel(st.BestOf)+" · "+rulesSummary(st.Rules), g.st(g.pal.dim))
		}
		drawCentered(g.screen, w/2, h-1, "Esc cancel", g.st(g.pal.dim))
		return
	}
	if rid := l.hostRoomID(); rid != "" {
		st := g.save.Settings
		rc := l.hostRoomCode()
		bh := 11
		if rc != "" {
			bh += 2
		}
		g.drawBox(bx, by, bw, bh, "Waiting for an opponent")
		drawCentered(g.screen, w/2, by+2, "Listed on Online Lobby ("+l.lobbyAddr()+")", g.st(g.pal.accent).Bold(true))
		drawCentered(g.screen, w/2, by+4, "Room ID: "+rid, g.st(g.pal.fg))
		row := by + 6
		if rc != "" {
			drawCentered(g.screen, w/2, row, "🔒 Room code: "+rc, g.st(g.pal.accent).Bold(true))
			row += 2
		}
		drawCentered(g.screen, w/2, row, "Waiting time: "+formatWait(time.Since(l.waitingSince())), g.st(g.pal.fg))
		drawCentered(g.screen, w/2, row+1, roundsLabel(st.BestOf)+" · "+rulesSummary(st.Rules), g.st(g.pal.dim))
		drawCentered(g.screen, w/2, h-1, "Esc cancel", g.st(g.pal.dim))
		return
	}
	targetAddr := l.lobbyAddr()
	if targetAddr == "" {
		targetAddr = g.save.Settings.JoinAddr
	}
	if e := l.lobbyError(); e != "" {
		lines := wrap(lobbyErrorText(e), bw-6)
		btnY := by + 2 + len(lines) + 3
		if btnY < by+7 {
			btnY = by + 7
		}
		bh := (btnY - by) + 3
		if bh > h-by-1 {
			bh = h - by - 1
			btnY = by + bh - 3
		}
		g.drawBox(bx, by, bw, bh, "Network Play")
		for i, line := range lines {
			drawCentered(g.screen, w/2, by+2+i, line, g.st(g.pal.lose))
		}
		addrY := btnY - 2
		if addrY > by+1+len(lines) {
			drawCentered(g.screen, w/2, addrY, targetAddr, g.st(g.pal.dim))
		}
		if !lobbyCanRetry(e) {
			st := tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
			drawCentered(g.screen, w/2, btnY, "▶ Back ◀", st)
			drawCentered(g.screen, w/2, h-1, "Enter / Esc back", g.st(g.pal.dim))
			return
		}
		backLabel := "  Back  "
		retryLabel := "  Retry  "
		backSt := g.st(g.pal.fg)
		retrySt := g.st(g.pal.fg)
		if g.lobbyBackFocus {
			backLabel = "▶ Back ◀"
			backSt = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
		} else {
			retryLabel = "▶ Retry ◀"
			retrySt = tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true)
		}
		total := textWidth(backLabel) + 4 + textWidth(retryLabel)
		x := w/2 - total/2
		x = drawText(g.screen, x, btnY, backLabel, backSt) + 4
		drawText(g.screen, x, btnY, retryLabel, retrySt)
		drawCentered(g.screen, w/2, h-1, "←→ select · Enter confirm · Esc back", g.st(g.pal.dim))
		return
	}
	g.drawBox(bx, by, bw, 9, "Network Play")
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/400%4))
	drawCentered(g.screen, w/2, by+2, "Waiting for game to start"+dots, g.st(g.pal.fg))
	drawCentered(g.screen, w/2, by+4, targetAddr, g.st(g.pal.accent))
	drawCentered(g.screen, w/2, by+6, "▶ Cancel ◀", tcell.StyleDefault.Background(g.pal.accent).Foreground(g.pal.bg).Bold(true))
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
	top, gx, topGridY, botGridY, topScoreY, botScoreY, midY, slotGap, gridH int
}

func (g *game) layout() playLayout {
	sg := slotGap
	if g.height < 27 {
		sg = 0
	}
	gh := gridRows*dieH + (gridRows-1)*sg
	top := 0
	if g.height > 28 {
		top = (g.height - 28) / 2
	}
	gx := (g.width - gridW) / 2
	if g.width < 68 {
		if alt := g.width - gridW - 2; alt >= 12 {
			gx = alt
		}
	}
	l := playLayout{top: top, gx: gx, slotGap: sg, gridH: gh}
	if g.height < 27 {
		l.topScoreY = top + 1
		l.topGridY = top + 2
	} else {
		l.topScoreY = top + 2
		l.topGridY = top + 3
	}
	l.midY = l.topGridY + gh
	l.botGridY = l.midY + 1
	l.botScoreY = l.botGridY + gh
	return l
}

func (l playLayout) cell(isTop bool, c, k int) (int, int) {
	x := l.gx + c*(dieW+colGap)
	if isTop {
		return x, l.topGridY + (gridRows-1-k)*(dieH+l.slotGap)
	}
	return x, l.botGridY + k*(dieH+l.slotGap)
}

func (g *game) drawGrid(l playLayout, grid *Grid, isTop, active bool, m gridMarks, ghostVal int) {
	if active && l.gx >= 2 {
		by := l.botGridY
		if isTop {
			by = l.topGridY
		}
		ast := g.st(g.pal.accent).Bold(true)
		for dy := 0; dy < l.gridH; dy++ {
			g.screen.SetContent(l.gx-1, by+dy, '┃', nil, ast)
		}
	}
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
				drawDieMarked(g.screen, x, y, v, g.cellMark(0, true), g.pal.doomBg, g.pal.doomPip)
			default:
				matches := grid.Matches(c, v)
				bg, pip := g.dieColors(matches)
				if c == m.lastCol && k == m.lastSlot {
					pip = g.pal.lastPip
				}
				drawDieMarked(g.screen, x, y, v, g.cellMark(matches, false), bg, pip)
			}
		}
	}
}

func (g *game) drawColumnScores(l playLayout, y int, grid, preview *Grid, changed tcell.Color, flashCol, flashDelta int, flashLose bool) {
	if y >= g.height {
		return
	}
	for c := 0; c < gridCols; c++ {
		x, _ := l.cell(true, c, 0)
		now := grid.ColumnScore(c)
		text, st := fmt.Sprint(now), g.st(g.pal.fg).Bold(true)
		if preview != nil {
			if next := preview.ColumnScore(c); next != now {
				text, st = fmt.Sprintf("%d→%d", now, next), g.st(changed).Bold(true)
			}
		} else if c == flashCol && flashDelta > 0 {
			if flashLose {
				text, st = fmt.Sprintf("%d -%d", now, flashDelta), g.st(g.pal.lose).Bold(true)
			} else {
				text, st = fmt.Sprintf("%d +%d", now, flashDelta), g.st(g.pal.win).Bold(true)
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

	bottom := g.bottomSeat()
	topP := 1 - bottom
	m := &s.Match
	lastPl := s.Last
	replayNote := ""
	if g.replayActive && len(g.replayFrames) > 0 {
		step := g.replayStep
		if step < 0 {
			step = 0
		}
		if step >= len(g.replayFrames) {
			step = len(g.replayFrames) - 1
		}
		rf := &g.replayFrames[step]
		m = &rf.match
		lastPl = rf.last
		replayNote = fmt.Sprintf("REPLAY %d/%d · %s", rf.step, rf.total, rf.desc)
	}

	// Preview for whoever is aiming: local cursor, or the opponent's shared aim.
	mover := m.Turn
	var preview *Match
	var pl Placement
	var mv Move
	var aiming bool
	if !g.replayActive {
		mv, aiming = g.previewMove(s)
		if aiming && m.Legal(mv) {
			p, next := m.Preview(mv)
			pl, preview = p, &next
		}
	}

	flashOn := !g.replayActive && effectProgress(g.effectAt, now) >= 0

	for _, p := range []int{topP, bottom} {
		isTop := p == topP
		active := !g.replayActive && s.Phase == phasePlay && p == mover
		marks := noMarks()
		grid := &m.Grids[p]
		ghostVal := 0
		if lastPl != nil && lastPl.Player == p && (s.Phase == phasePlay || g.replayActive) {
			marks.lastCol, marks.lastSlot = lastPl.Col, lastPl.Slot
			if marks.lastSlot >= 0 && grid[marks.lastCol][marks.lastSlot] != lastPl.Die {
				marks.lastCol = -1
			}
		}
		flashCol, flashDelta, flashLose := -1, 0, false
		if flashOn {
			if p == g.effectLast.Player && g.effectLast.Gain > 0 {
				flashCol, flashDelta = g.effectLast.Col, g.effectLast.Gain
			} else if p != g.effectLast.Player && g.effectLast.Destroyed > 0 && g.effectLast.Lost > 0 {
				flashCol, flashDelta, flashLose = g.effectLast.Col, g.effectLast.Lost, true
			}
		}
		var scorePreview *Grid
		var scoreColor tcell.Color
		switch {
		case preview != nil && p == mover:
			marks.ghostCol, marks.ghostSlot = pl.Col, pl.Slot
			ghostVal = pl.Die
			scorePreview, scoreColor = &preview.Grids[p], g.pal.win
			g.drawGrid(l, &preview.Grids[p], isTop, active, marks, ghostVal)
		case preview != nil:
			if !m.Rules.NoDestroy {
				marks.doomCol, marks.doomVal = pl.Col, pl.Die
			}
			scorePreview, scoreColor = &preview.Grids[p], g.pal.lose
			g.drawGrid(l, grid, isTop, active, marks, 0)
		default:
			g.drawGrid(l, grid, isTop, active, marks, 0)
		}
		y := l.botScoreY
		if isTop {
			y = l.topScoreY
		}
		g.drawColumnScores(l, y, grid, scorePreview, scoreColor, flashCol, flashDelta, flashLose)
	}

	if replayNote != "" {
		drawCentered(g.screen, l.gx+gridW/2, l.midY, truncate(replayNote, w-2), g.st(g.pal.accent).Bold(true))
	} else if aiming {
		x, _ := l.cell(true, mv.Col, 0)
		arrow := "▼"
		if mover == topP {
			arrow = "▲"
		}
		mst := g.st(g.pal.accent).Bold(true)
		if !m.Grids[mover].Open(mv.Col) {
			mst = g.st(g.pal.dim)
		}
		drawCentered(g.screen, x+dieW/2, l.midY, arrow+" "+fmt.Sprint(mv.Col+1)+" "+arrow, mst)
	}

	g.drawHeader(s, l, now)
	g.drawPlayerPanel(s, m, l, topP, true, pl, preview != nil, now)
	g.drawPlayerPanel(s, m, l, bottom, false, pl, preview != nil, now)
	g.drawSidePanel(s, l)

	keys := "←→ col · 1-3/Enter · C · P · S snd · L log · H · R · Esc"
	if m.Rules.TwoDice {
		keys = "←→ col · Tab/W die · Enter · C · P · S · L · H · R · Esc"
	}
	if g.replayActive {
		keys = "←→ step · Home/End · L log · H hist · Esc/X done"
	} else if s.Phase != phasePlay {
		keys = "←→ choose · Enter · V view board · L log · H hist · Esc leave"
		if canReplay(s) {
			keys = "←→ choose · Enter · V board · X replay · L log · H · Esc"
		}
	}
	if h-1 > l.botScoreY || h >= 24 {
		drawCentered(g.screen, w/2, h-1, truncate(keys, w-2), g.st(g.pal.dim))
	}

	switch {
	case g.overlay == overlayLog:
		g.drawLogOverlay(s)
	case g.overlay == overlayHistory:
		g.drawHistoryOverlay(s)
	case g.overlay == overlayResign:
		g.drawConfirm("Resign this match?", g.resignWarning(s), "Cancel", "Confirm resignation")
	case g.overlay == overlayLeave:
		g.drawConfirm("Leave this game?", leaveWarning(s), "Stay", "Leave")
	case s.Phase != phasePlay && !g.hideResult && !g.replayActive:
		g.drawResult(s, l)
	}
}

func (g *game) drawHeader(s *session, l playLayout, now time.Time) {
	w := g.width
	y := l.top
	text, c := g.headerText(s, now)
	if g.replayActive {
		text, c = "Round Replay", g.pal.accent
	}
	drawText(g.screen, 1, y, "KNUCKLEBONES", g.st(g.pal.title).Bold(true))
	drawCentered(g.screen, w/2, y, text, g.st(c).Bold(true))
	label := fmt.Sprintf("Round %d · %s", s.Round, s.roundsLabel())
	drawText(g.screen, w-1-textWidth(label), y, label, g.st(g.pal.dim))
}

func (g *game) drawPlayerPanel(s *session, m *Match, l playLayout, p int, isTop bool, pl Placement, hasPreview bool, now time.Time) {
	pw := l.gx - 3
	if pw < 10 {
		return
	}
	x := 1
	turn := !g.replayActive && s.Phase == phasePlay && m.Turn == p

	var nameY, trayY int
	if isTop {
		nameY, trayY = l.topGridY, l.topGridY+l.gridH-dieH-1
		if trayY < nameY+3 {
			trayY = nameY + 3
		}
	} else {
		trayY, nameY = l.botGridY+1, l.botGridY+l.gridH-4
		if l.gridH < 10 {
			trayY = l.botGridY
			nameY = l.botGridY + l.gridH - 3
		}
	}

	nameSt := g.st(g.pal.fg).Bold(true)
	if turn {
		nameSt = g.st(g.pal.accent).Bold(true)
	}
	drawText(g.screen, x, nameY, truncate(g.panelName(s, p), pw), nameSt)
	drawText(g.screen, x, nameY+1, "Total ", g.st(g.pal.dim))
	drawText(g.screen, x+6, nameY+1, fmt.Sprint(m.Grids[p].Score()), g.st(g.pal.fg).Bold(true))

	if info := g.winsText(s, p); info != "" && nameY+2 < g.height-1 {
		drawText(g.screen, x, nameY+2, truncate(info, pw), g.st(g.pal.dim))
	}
	if !isTop && g.save.Settings.WinMeter && !g.replayActive && g.width-(l.gx+gridW+3)-1 < 14 && nameY+3 < g.height-1 {
		_, wpLabel := g.winProbSummary(s)
		drawText(g.screen, x, nameY+3, truncate("Odds: "+wpLabel, pw), g.st(g.pal.win))
	}
	if s.TurnLimit > 0 && s.Timeouts[p] > 0 && nameY+3 < g.height-1 {
		drawText(g.screen, x, nameY+3, truncate(fmt.Sprintf("Timeouts: %d / %d", s.Timeouts[p], maxTimeouts), pw), g.st(g.pal.lose))
	}

	if !turn {
		return
	}
	rolling := g.rolling()
	selDie := g.dieSel
	if !g.myTurn(s) && s.Aiming {
		selDie = s.AimDie
	}
	for i := 0; i < 2; i++ {
		v := m.Dice[i]
		if v == 0 {
			continue
		}
		dx := x + i*(dieW+2)
		if dx+dieW > l.gx-1 {
			continue
		}
		if rolling {
			drawDie(g.screen, dx, trayY, rollingFace(i), g.pal.dieBg, g.pal.pip)
			continue
		}
		bg, pip := g.pal.dieBg, g.pal.pip
		if m.Dice[1] != 0 && i != selDie {
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
		drawText(g.screen, x, ty, truncate(gainText(pl), pw), g.st(g.pal.accent))
	case !g.myTurn(s) && !rolling:
		drawText(g.screen, x, ty, thinkingText(now), g.st(g.pal.dim))
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
	if ft, fc := g.fairnessText(s); ft != "" {
		drawText(g.screen, x, y, truncate(ft, pw), g.st(fc))
		y += 2
	}
	if g.save.Settings.WinMeter && !g.replayActive {
		pct, label := g.winProbSummary(s)
		drawText(g.screen, x, y, truncate("Win odds · "+label, pw), g.st(g.pal.accent))
		y++
		barW := 12
		if barW > pw-2 {
			barW = pw - 2
		}
		if barW >= 4 {
			filled := pct * barW / 100
			drawText(g.screen, x, y, strings.Repeat("█", filled), g.st(g.pal.win))
			drawText(g.screen, x+filled, y, strings.Repeat("░", barW-filled), g.st(g.pal.dim))
			y += 2
		}
	}
	drawText(g.screen, x, y, "Log", g.st(g.pal.accent).Bold(true))
	y++
	// Newest entries first, drawn top-down; the newest entry is brighter.
	room := l.botScoreY - y
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
	if l.botScoreY < g.height-1 {
		sc := g.pal.dim
		if g.save.Settings.Sound {
			sc = g.pal.accent
		}
		drawText(g.screen, x, l.botScoreY, truncate("Sound: "+onOff(g.save.Settings.Sound)+" [S]", pw), g.st(sc))
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
	ft, fc := g.fairnessText(s)
	extraH := 0
	if ft != "" {
		extraH = 1
	}
	bh := 9 + len(lines) + extraH
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
	if status, stc := g.resultStatus(s); status != "" {
		drawCentered(g.screen, w/2, y, status, g.st(stc))
	}
	if ft != "" {
		y++
		drawCentered(g.screen, w/2, y, ft, g.st(fc))
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
		c := g.logColor(e)
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
	for _, l := range g.historyLines(s) {
		lines = append(lines, l.text)
		colors = append(colors, l.color)
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
	{"Network play", 1},
	{"Open Network Play to host or join via Online Lobby (vps.kreft.us) or Direct connect. Locked rooms (🔒) require the host's room code. The host rolls the dice; turns can be timed and three timeouts lose the match. A dropped player has 30 seconds to reconnect.", 0},
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
	lines, kinds := g.guideLines(func(t string) []string { return wrap(t, tw-2) })
	styles := make([]tcell.Style, len(lines))
	for i, k := range kinds {
		styles[i] = g.st(g.pal.fg)
		if k == 1 {
			styles[i] = g.st(g.pal.accent).Bold(true)
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
	format := func(c [7]string) string {
		return fmt.Sprintf("%-22s %6s %5s %7s %6s %5s %7s", c[0], c[1], c[2], c[3], c[4], c[5], c[6])
	}
	drawText(g.screen, x, y, format(statsHeader), g.st(g.pal.dim))
	y++
	for _, r := range g.statsRows() {
		c := g.pal.fg
		if r.empty {
			c = g.pal.dim
		}
		drawText(g.screen, x, y, format(r.cells), g.st(c))
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
