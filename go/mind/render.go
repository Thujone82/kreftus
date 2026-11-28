package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	boardW = 66
	boardH = 21
	boxH   = 20
	splitX = 40
)

type cell struct {
	ch   rune
	fg   rgb
	bg   rgb
	bold bool
}

type canvas struct {
	w, h  int
	cells [][]cell
}

func newCanvas(w, h int, bg, fg rgb) *canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	cells := make([][]cell, h)
	for y := 0; y < h; y++ {
		row := make([]cell, w)
		for x := 0; x < w; x++ {
			row[x] = cell{ch: ' ', fg: fg, bg: bg}
		}
		cells[y] = row
	}
	return &canvas{w: w, h: h, cells: cells}
}

func (c *canvas) set(x, y int, ch rune, fg, bg rgb, bold bool) {
	if x < 0 || x >= c.w || y < 0 || y >= c.h {
		return
	}
	c.cells[y][x] = cell{ch: ch, fg: fg, bg: bg, bold: bold}
}

func (c *canvas) setFG(x, y int, ch rune, fg rgb, bold bool) {
	if x < 0 || x >= c.w || y < 0 || y >= c.h {
		return
	}
	c.cells[y][x].ch = ch
	c.cells[y][x].fg = fg
	c.cells[y][x].bold = bold
}

func (c *canvas) fillRect(x, y, w, h int, fg, bg rgb) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.set(xx, yy, ' ', fg, bg, false)
		}
	}
}

func (c *canvas) text(x, y int, s string, fg rgb, bold bool) {
	col := x
	for _, r := range s {
		if col >= c.w {
			break
		}
		if col >= 0 {
			c.setFG(col, y, r, fg, bold)
		}
		col++
	}
}

func (c *canvas) textCentered(y int, s string, fg rgb, bold bool) {
	runes := []rune(s)
	x := (c.w - len(runes)) / 2
	c.text(x, y, s, fg, bold)
}

func (c *canvas) box(x, y, w, h int, bs boxStyle, fg rgb) {
	if w < 2 || h < 2 {
		return
	}
	c.setFG(x, y, bs.TL, fg, false)
	c.setFG(x+w-1, y, bs.TR, fg, false)
	c.setFG(x, y+h-1, bs.BL, fg, false)
	c.setFG(x+w-1, y+h-1, bs.BR, fg, false)
	for xx := x + 1; xx < x+w-1; xx++ {
		c.setFG(xx, y, bs.H, fg, false)
		c.setFG(xx, y+h-1, bs.H, fg, false)
	}
	for yy := y + 1; yy < y+h-1; yy++ {
		c.setFG(x, yy, bs.V, fg, false)
		c.setFG(x+w-1, yy, bs.V, fg, false)
	}
}

func (c *canvas) hline(x, y, w int, left, mid, right rune, fg rgb) {
	if w < 2 {
		return
	}
	c.setFG(x, y, left, fg, false)
	for xx := x + 1; xx < x+w-1; xx++ {
		c.setFG(xx, y, mid, fg, false)
	}
	c.setFG(x+w-1, y, right, fg, false)
}

func (c *canvas) vline(x, y, h int, line rune, fg rgb) {
	for yy := y; yy < y+h; yy++ {
		c.setFG(x, yy, line, fg, false)
	}
}

func contrastText(bg rgb) rgb {
	// Relative luminance approximation (0..255).
	lum := (299*int(bg.R) + 587*int(bg.G) + 114*int(bg.B)) / 1000
	if lum > 140 {
		return rgb{0, 0, 0}
	}
	return rgb{255, 255, 255}
}

// drawPeg draws a 3-column peg slot starting at x.
func (c *canvas) drawPeg(x, y int, colorIdx int, th theme, labels, numbers, bold bool) {
	if colorIdx < 0 || colorIdx >= numColors {
		return
	}
	pcol := th.Pegs[colorIdx]
	if th.Tiles || labels {
		fg := contrastText(pcol)
		ch := rune(colors[colorIdx])
		if numbers {
			ch = rune('1' + colorIdx)
		}
		c.set(x, y, ' ', fg, pcol, bold)
		c.set(x+1, y, ch, fg, pcol, true)
		c.set(x+2, y, ' ', fg, pcol, bold)
		return
	}
	c.setFG(x+1, y, '⬤', pcol, bold)
}

func (c *canvas) drawTitleText(y, offset int, th theme) {
	const title = "M A S T E R M I N D"
	x := (c.w - len([]rune(title))) / 2
	ci := offset
	for _, r := range title {
		if r != ' ' {
			c.setFG(x, y, r, th.Pegs[ci%numColors], true)
			ci++
		}
		x++
	}
}

func renderBoard(g *gameState, st settings, th theme, now time.Time) *canvas {
	c := newCanvas(boardW, boardH, th.BG, th.FG)

	frameCol := th.Frame
	titleOffset := 0
	if g.celebrateStep >= 0 {
		frameCol = th.Pegs[g.celebrateStep%numColors]
		titleOffset = g.celebrateStep
	}

	c.box(0, 0, boardW, boxH, th.Box, frameCol)
	c.drawTitleText(1, titleOffset, th)

	// Row 2: full-width divider with tee down to the right panel.
	c.hline(0, 2, boardW, th.Box.LT, th.Box.H, th.Box.RT, frameCol)
	c.setFG(splitX, 2, th.Box.ITT, frameCol, false)

	// Vertical divider between board (0..40) and side panel (40..65).
	c.vline(splitX, 3, boxH-4, th.Box.IV, frameCol)
	c.setFG(splitX, boxH-1, th.Box.IBT, frameCol, false)

	// Row 3: CODE row.
	c.text(2, 3, "CODE", th.Dim, true)
	for i := 0; i < codeLength; i++ {
		px := 7 + i*4
		if g.over && i < g.revealedPegs {
			c.drawPeg(px, 3, colorIndex(g.secret[i]), th, st.Labels, st.LabelNumbers, true)
		} else {
			c.setFG(px+1, 3, th.Cover, th.Accent, false)
		}
	}

	// Row 4 and Row 17: left-board inner separators.
	c.hline(0, 4, splitX+1, th.Box.ILT, th.Box.IH, '┤', frameCol)
	c.hline(0, 17, splitX+1, th.Box.ILT, th.Box.IH, '┤', frameCol)

	// Rows 5..16: Turns 01..12.
	activeTurn := len(g.turns)
	for i := 0; i < maxTurns; i++ {
		y := 5 + i
		numStr := fmt.Sprintf("%02d", i+1)

		switch {
		case i < len(g.turns):
			tr := g.turns[i]
			c.text(2, y, numStr, th.Dim, false)
			c.setFG(5, y, th.Box.IV, frameCol, false)
			for p := 0; p < codeLength; p++ {
				c.drawPeg(7+p*4, y, colorIndex(tr.Guess[p]), th, st.Labels, st.LabelNumbers, false)
			}
			c.setFG(23, y, th.Box.IV, frameCol, false)

			shown := codeLength
			if g.animFeedback && i == len(g.turns)-1 {
				shown = g.animShown
			}
			drawFeedback(c, 25, y, tr.Exact, tr.Partial, shown, th)

		case i == activeTurn && !g.over:
			c.setFG(1, y, '▶', th.Accent, true)
			c.text(2, y, numStr, th.Accent, true)
			c.setFG(5, y, th.Box.IV, frameCol, false)
			for p := 0; p < codeLength; p++ {
				px := 7 + p*4
				if p < len(g.current) {
					c.drawPeg(px, y, colorIndex(g.current[p]), th, st.Labels, st.LabelNumbers, true)
				} else {
					c.setFG(px+1, y, th.EmptySlot, th.Dim, false)
				}
			}
			c.setFG(23, y, th.Box.IV, frameCol, false)
			drawFeedback(c, 25, y, 0, 0, codeLength, th)

		default:
			c.text(2, y, numStr, th.Dim, false)
			c.setFG(5, y, th.Box.IV, frameCol, false)
			for p := 0; p < codeLength; p++ {
				c.setFG(7+p*4+1, y, th.EmptySlot, th.Dim, false)
			}
			c.setFG(23, y, th.Box.IV, frameCol, false)
			drawFeedback(c, 25, y, 0, 0, codeLength, th)
		}
	}

	// Row 18: Status line.
	statusText := "Enter 4 pegs, then Enter"
	statusColor := th.Accent
	if g.statusMsg != "" && now.Before(g.statusUntil) {
		statusText = g.statusMsg
		if g.statusIsErr {
			statusColor = th.Lose
		}
	} else if g.over {
		if g.won {
			statusText = "Code cracked! N new game"
			statusColor = th.Win
		} else {
			statusText = "Out of turns. N new game"
			statusColor = th.Lose
		}
	} else if g.animFeedback {
		statusText = "Checking code..."
	} else if len(g.current) == codeLength {
		statusText = "Press Enter to submit guess"
	}
	c.text(2, 18, statusText, statusColor, true)

	// Right panel (cols 42..64, rows 3..18).
	dispTurn := len(g.turns) + 1
	if g.over && dispTurn > len(g.turns) {
		dispTurn = len(g.turns)
	}
	if dispTurn > maxTurns {
		dispTurn = maxTurns
	}
	c.text(42, 3, "Turn", th.Dim, false)
	c.text(47, 3, fmt.Sprintf("%02d / %02d", dispTurn, maxTurns), th.FG, true)

	c.text(42, 4, "Time", th.Dim, false)
	c.text(47, 4, formatClock(g.elapsed()), th.FG, true)

	c.text(42, 6, "PEGS", th.Accent, true)
	for i := 0; i < numColors; i++ {
		y := 7 + i
		if th.Tiles || st.Labels {
			c.drawPeg(42, y, i, th, st.Labels, st.LabelNumbers, false)
			if st.LabelNumbers {
				c.text(46, y, fmt.Sprintf("%c %s", colors[i], colorNames[i]), th.FG, false)
			} else {
				c.text(46, y, fmt.Sprintf("%d %s", i+1, colorNames[i]), th.FG, false)
			}
		} else {
			c.setFG(42, y, '⬤', th.Pegs[i], false)
			letterCol, letterBold := th.Accent, true
			numCol, numBold := th.Dim, false
			if st.LabelNumbers {
				letterCol, letterBold = th.Dim, false
				numCol, numBold = th.Accent, true
			}
			c.setFG(44, y, rune(colors[i]), letterCol, letterBold)
			c.setFG(46, y, rune('1'+i), numCol, numBold)
			c.text(48, y, colorNames[i], th.FG, false)
		}
	}

	c.text(42, 14, "FEEDBACK", th.Accent, true)
	c.setFG(42, 15, th.ExactGlyph, th.Exact, true)
	c.text(44, 15, "right spot", th.FG, false)
	c.setFG(42, 16, th.PartialGlyph, th.Partial, true)
	c.text(44, 16, "right color", th.FG, false)

	c.text(42, 18, fmt.Sprintf("Wins %d/%d  Streak %d", st.Stats.Won, st.Stats.Played, st.Stats.Streak), th.Dim, false)

	// Row 20: Footer shortcuts.
	drawFooter(c, 20, st, th)

	if g.overlayVisible {
		drawOverlay(c, g, th)
	}

	return c
}

func drawFeedback(c *canvas, x, y, exact, partial, shown int, th theme) {
	for i := 0; i < codeLength; i++ {
		fx := x + i*2
		if i >= shown {
			c.setFG(fx, y, th.EmptyFbGlyph, th.Dim, false)
			continue
		}
		if i < exact {
			c.setFG(fx, y, th.ExactGlyph, th.Exact, true)
		} else if i < exact+partial {
			c.setFG(fx, y, th.PartialGlyph, th.Partial, false)
		} else {
			c.setFG(fx, y, th.EmptyFbGlyph, th.Dim, false)
		}
	}
}

func labelStateText(st settings, th theme) string {
	if !th.Tiles && !st.Labels {
		return "Off"
	}
	if st.LabelNumbers {
		return "123"
	}
	return "RGB"
}

func drawFooter(c *canvas, y int, st settings, th theme) {
	soundLabel := "Off"
	if st.Sound {
		soundLabel = "On"
	}

	parts := []struct {
		key string
		val string
	}{
		{"T", " Theme: " + th.Name + "  "},
		{"S", " Sound ♪ " + soundLabel + "  "},
		{"L/Tab", " Labels " + labelStateText(st, th) + "  "},
		{"Esc", " Quit"},
	}

	totalLen := 0
	for _, p := range parts {
		totalLen += len([]rune(p.key)) + len([]rune(p.val))
	}
	x := (c.w - totalLen) / 2
	if x < 0 {
		x = 0
	}
	for _, p := range parts {
		c.text(x, y, p.key, th.Accent, true)
		x += len([]rune(p.key))
		c.text(x, y, p.val, th.Dim, false)
		x += len([]rune(p.val))
	}
}

func drawOverlay(c *canvas, g *gameState, th theme) {
	const ow, oh = 40, 7
	ox := (c.w - ow) / 2
	oy := (boxH - oh) / 2

	borderCol := th.Lose
	headline := "OUT OF TURNS"
	detail := fmt.Sprintf("Secret revealed · %s", formatClock(g.elapsed()))
	if g.won {
		borderCol = th.Win
		headline = "★ CODE CRACKED ★"
		detail = fmt.Sprintf("Solved in %d turns · %s", len(g.turns), formatClock(g.elapsed()))
	}

	c.fillRect(ox, oy, ow, oh, th.FG, th.BG)
	c.box(ox, oy, ow, oh, th.Box, borderCol)
	c.textCentered(oy+2, headline, borderCol, true)
	c.textCentered(oy+3, detail, th.FG, false)

	prompt := "N New game   Esc Quit"
	px := (c.w - len([]rune(prompt))) / 2
	c.text(px, oy+4, "N", th.Accent, true)
	c.text(px+1, oy+4, " New game   ", th.Dim, false)
	c.text(px+13, oy+4, "Esc", th.Accent, true)
	c.text(px+16, oy+4, " Quit", th.Dim, false)
}

func renderTitleScreen(st settings, th theme, animTick int) *canvas {
	c := newCanvas(boardW, boardH, th.BG, th.FG)
	c.box(0, 0, boardW, boxH, th.Box, th.Frame)
	c.drawTitleText(1, 0, th)
	c.hline(0, 2, boardW, th.Box.LT, th.Box.H, th.Box.RT, th.Frame)

	// Centered row of 6 animated pegs.
	active := animTick % numColors
	startX := (boardW - (numColors*4 - 1)) / 2
	for i := 0; i < numColors; i++ {
		px := startX + i*4
		c.drawPeg(px, 4, i, th, st.Labels, st.LabelNumbers, i == active)
		if i == active {
			c.setFG(px-1, 4, '▶', th.Accent, true)
		}
	}

	c.textCentered(6, "Crack the secret 4-peg code in 12 turns.", th.FG, false)
	c.textCentered(7, "Colors: R G B C M Y  (or number keys 1-6)", th.Dim, false)

	fbLine := fmt.Sprintf("%c right spot    %c right color", th.ExactGlyph, th.PartialGlyph)
	fx := (boardW - len([]rune(fbLine))) / 2
	c.setFG(fx, 8, th.ExactGlyph, th.Exact, true)
	c.text(fx+1, 8, " right spot    ", th.FG, false)
	c.setFG(fx+16, 8, th.PartialGlyph, th.Partial, true)
	c.text(fx+17, 8, " right color", th.FG, false)

	c.hline(0, 10, boardW, th.Box.ILT, th.Box.IH, th.Box.IRT, th.Frame)

	soundState := "♪ Off"
	if st.Sound {
		soundState = "♪ On"
	}
	labelsState := "Off"
	if st.Labels {
		labelsState = "On"
	}

	labelStyle := "RGB"
	if st.LabelNumbers {
		labelStyle = "123"
	}

	menuX := 20
	c.text(menuX, 12, "Enter", th.Accent, true)
	c.text(menuX+8, 12, "Play", th.FG, true)

	c.text(menuX, 13, "T", th.Accent, true)
	c.text(menuX+8, 13, "Theme   < "+th.Name+" >", th.FG, false)

	c.text(menuX, 14, "S", th.Accent, true)
	c.text(menuX+8, 14, "Sound   "+soundState, th.FG, false)

	c.text(menuX, 15, "L", th.Accent, true)
	c.text(menuX+8, 15, "Labels  "+labelsState, th.FG, false)

	c.text(menuX, 16, "Tab", th.Accent, true)
	c.text(menuX+8, 16, "Label style  "+labelStyle, th.FG, false)

	c.text(menuX, 17, "Esc", th.Accent, true)
	c.text(menuX+8, 17, "Quit", th.FG, false)

	var statLine string
	if st.Stats.Played == 0 {
		statLine = "No games played yet - good luck!"
	} else {
		bestT := "-"
		if st.Stats.BestTurns > 0 {
			bestT = fmt.Sprintf("%d turns", st.Stats.BestTurns)
		}
		bestTime := "-"
		if st.Stats.BestTimeSec > 0 {
			bestTime = formatClock(time.Duration(st.Stats.BestTimeSec) * time.Second)
		}
		statLine = fmt.Sprintf("Played %d · Won %d · Best %s · Fastest %s", st.Stats.Played, st.Stats.Won, bestT, bestTime)
	}
	c.textCentered(18, statLine, th.Dim, false)
	c.textCentered(20, "Press Enter to start  ·  T to switch themes", th.Dim, false)

	return c
}

func renderTooSmall(termW, termH int, th theme) *canvas {
	c := newCanvas(termW, termH, th.BG, th.FG)
	msg := fmt.Sprintf("Enlarge window to %dx%d (now %dx%d)", boardW, boardH, termW, termH)
	c.textCentered(termH/2, msg, th.Accent, true)
	return c
}

// serializeCanvas renders a canvas into an ANSI true-color string.
// When fullScreen is true, it centers the canvas inside termW x termH using \033[H and \033[J.
func serializeCanvas(c *canvas, termW, termH int, bg rgb, fullScreen bool) string {
	var b strings.Builder
	b.Grow(c.w * c.h * 12)

	if fullScreen {
		b.WriteString("\033[H")
	}

	leftPad := 0
	topPad := 0
	if fullScreen {
		if termW > c.w {
			leftPad = (termW - c.w) / 2
		}
		if termH > c.h {
			topPad = (termH - c.h) / 2
		}
	}

	bgSeq := fmt.Sprintf("\033[48;2;%d;%d;%dm", bg.R, bg.G, bg.B)

	if fullScreen && topPad > 0 {
		b.WriteString(bgSeq)
		for y := 0; y < topPad; y++ {
			b.WriteString("\033[K\r\n")
		}
	}

	var (
		hasState bool
		curFG    rgb
		curBG    rgb
		curBold  bool
	)

	for y := 0; y < c.h; y++ {
		if leftPad > 0 {
			if !hasState || curBG != bg || curBold {
				b.WriteString("\033[0m")
				b.WriteString(bgSeq)
				curBG = bg
				curBold = false
				hasState = false
			}
			for i := 0; i < leftPad; i++ {
				b.WriteByte(' ')
			}
		}

		for x := 0; x < c.w; x++ {
			cl := c.cells[y][x]
			if !hasState || cl.bold != curBold {
				if !cl.bold && hasState {
					b.WriteString("\033[0m")
					hasState = false
				}
			}
			if !hasState || cl.bg != curBG {
				fmt.Fprintf(&b, "\033[48;2;%d;%d;%dm", cl.bg.R, cl.bg.G, cl.bg.B)
				curBG = cl.bg
			}
			if !hasState || cl.fg != curFG {
				fmt.Fprintf(&b, "\033[38;2;%d;%d;%dm", cl.fg.R, cl.fg.G, cl.fg.B)
				curFG = cl.fg
			}
			if cl.bold && (!hasState || !curBold) {
				b.WriteString("\033[1m")
				curBold = true
			}
			hasState = true
			b.WriteRune(cl.ch)
		}

		if !hasState || curBG != bg || curBold {
			b.WriteString("\033[0m")
			b.WriteString(bgSeq)
			curBG = bg
			curBold = false
			hasState = false
		}
		if fullScreen {
			b.WriteString("\033[K")
			if y+1 < c.h {
				b.WriteString("\r\n")
			}
		} else {
			b.WriteString("\033[0m\n")
			hasState = false
		}
	}

	if fullScreen {
		b.WriteString(bgSeq)
		b.WriteString("\033[J")
	}

	return b.String()
}

func flushCanvas(c *canvas, termW, termH int, bg rgb) {
	out := serializeCanvas(c, termW, termH, bg, true)
	_, _ = os.Stdout.WriteString(out)
}

func renderPreviewString() string {
	var b strings.Builder
	sampleTurns := []turnRecord{
		{Guess: []byte("RRGG"), Exact: 1, Partial: 0},
		{Guess: []byte("BBGG"), Exact: 1, Partial: 1},
		{Guess: []byte("CCGB"), Exact: 3, Partial: 0},
	}
	sampleSecret := []byte("MCGB")

	for _, id := range themeOrder {
		th := themeByID(id)
		st := defaultSettings()
		st.Theme = id
		st.Stats = stats{Played: 8, Won: 5, Streak: 2, BestStreak: 3, BestTurns: 4, BestTimeSec: 27}

		fmt.Fprintf(&b, "=== Theme: %s (%s) - Mid-Game ===\n", th.Name, th.ID)
		mid := newGameState(sampleSecret)
		mid.turns = append([]turnRecord(nil), sampleTurns...)
		mid.current = []byte("MC")
		mid.startTime = time.Now().Add(-42 * time.Second)
		cMid := renderBoard(mid, st, th, time.Now())
		b.WriteString(serializeCanvas(cMid, boardW, boardH, th.BG, false))

		fmt.Fprintf(&b, "=== Theme: %s (%s) - Won ===\n", th.Name, th.ID)
		win := newGameState(sampleSecret)
		win.turns = append(append([]turnRecord(nil), sampleTurns...), turnRecord{Guess: []byte("MCGB"), Exact: 4, Partial: 0})
		win.startTime = time.Now().Add(-27 * time.Second)
		win.endTime = time.Now()
		win.won = true
		win.over = true
		win.revealedPegs = codeLength
		win.overlayVisible = true
		cWin := renderBoard(win, st, th, time.Now())
		b.WriteString(serializeCanvas(cWin, boardW, boardH, th.BG, false))
	}
	return b.String()
}
