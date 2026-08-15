package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const boardCols = 37
const boardRows = 19
const mistakeMark = '×' // same glyph as README 9×9
const pencilGlyph = '▀'
const modePen = "✒️"
const modePencil = "✏️"
const markRowLeft = '▶'
const markRowRight = '◀'
const markColDown = '▼'
const markColUp = '▲'

// Nine hues equally spaced around the wheel. White is reserved for a digit
// whose nine correct placements are all on the board.
var digitColor = [10]tcell.Color{
	tcell.ColorDefault,
	tcell.NewRGBColor(232, 64, 64),  // 1 red
	tcell.NewRGBColor(255, 140, 32), // 2 orange
	tcell.NewRGBColor(240, 210, 32), // 3 gold
	tcell.NewRGBColor(48, 196, 64),  // 4 green
	tcell.NewRGBColor(32, 204, 168), // 5 teal
	tcell.NewRGBColor(32, 168, 232), // 6 azure
	tcell.NewRGBColor(72, 104, 255), // 7 blue
	tcell.NewRGBColor(168, 80, 255), // 8 violet
	tcell.NewRGBColor(232, 64, 168), // 9 magenta
}

var digitCompleteColor = tcell.ColorWhite

var (
	styleDefault    = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	styleDim        = tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
	styleGrid       = tcell.StyleDefault.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack)
	styleGridPencil = tcell.StyleDefault.Foreground(tcell.NewRGBColor(255, 230, 120)).Background(tcell.ColorBlack)
)

func (g *game) render() {
	g.maybeShowSolved()
	s := g.screen
	s.Fill(' ', styleDefault)
	switch g.view {
	case viewMenu:
		g.drawMenu()
	case viewPlay:
		g.drawPlay()
	case viewPaused:
		g.drawPause()
	case viewConfirmNewGame:
		g.drawMenu()
		g.drawConfirm("Abandon the current game?", "This counts as a Failure.", "Cancel", "Abandon", false)
	case viewSolved:
		g.drawSolved()
	}
	s.Show()
}

func (g *game) drawMenu() {
	w, h := g.width, g.height
	g.drawSudokuBanner(1)

	done := g.save.completedSet(g.difficulty)
	remain := remainingCount(g.difficulty, done)
	canNew := remain > 0
	st := g.save.statsFor(g.difficulty)
	showDetail := st.Successes > 0 || len(g.save.Completed[g.difficulty]) > 0

	y := 3
	drawCentered(g.screen, w/2, y, "── "+difficultyLabel[g.difficulty]+" ──", g.titleStyle())
	y++

	visible := g.visibleMenu()
	labels := map[int]string{
		menuContinue: "Continue",
		menuNewGame:  "New Game",
		menuQuit:     "Quit",
	}
	for row, id := range visible {
		label := labels[id]
		itemSt := styleDefault
		disabled := id == menuNewGame && !canNew
		if disabled {
			itemSt = styleDim
			label = "New Game  (complete)"
		}
		if id == g.menuIndex {
			if disabled {
				itemSt = styleDim.Reverse(true)
			} else {
				itemSt = g.selectStyle()
			}
			label = " ▶ " + label + " ◀ "
		}
		drawCentered(g.screen, w/2, y+row, label, itemSt)
	}
	y += len(visible) + 1

	drawCentered(g.screen, w/2, y, "── Stats ──", g.titleStyle())
	y++
	statRows := [][2]string{}
	if showDetail {
		fast := "—"
		if st.FastestMs != nil {
			fast = formatDuration(time.Duration(*st.FastestMs) * time.Millisecond)
			if st.FastestMistakes != nil {
				fast += fmt.Sprintf("  (%d incorrect)", *st.FastestMistakes)
			}
		}
		statRows = [][2]string{
			{"Perfect", fmt.Sprintf("%d", st.Perfect)},
			{"Successes", fmt.Sprintf("%d", st.Successes)},
			{"Error Rate", st.errorRate()},
			{"Failed", fmt.Sprintf("%d", st.Failed)},
			{"Fastest", fast},
			{"Average Completion", g.save.averageCompletion(g.difficulty)},
		}
	}
	total := len(puzzlesFor(g.difficulty))
	statRows = append(statRows, [2]string{"Remaining", fmt.Sprintf("%d / %d", remain, total)})
	drawStatBlock(g.screen, w/2, y, formatStatLines(statRows), styleDefault)

	if h > 2 {
		drawCentered(g.screen, w/2, h-2, "↑↓ select · ←→ difficulty · Enter · Esc quit", styleDim)
	}
}

func (g *game) drawSudokuBanner(y int) {
	letters := []rune("SUDOKU")
	width := len(letters)*2 - 1
	x := g.width/2 - width/2
	if x < 0 {
		x = 0
	}
	for i, ch := range letters {
		if i > 0 {
			x++
		}
		st := tcell.StyleDefault.Foreground(g.wheelColor(5 + i)).Background(tcell.ColorBlack).Bold(true)
		g.screen.SetContent(x, y, ch, nil, st)
		x++
	}
}

func (g *game) drawPlay() {
	w, h := g.width, g.height
	g.drawHUD()
	ox := (w - boardCols) / 2
	if ox < 1 && w >= boardCols+2 {
		ox = 1
	}
	if ox < 0 {
		ox = 0
	}
	oy := 2
	g.drawBoard(ox, oy)
	activeY := oy + boardRows + 1
	if activeY < h {
		g.drawActiveLine(activeY)
	}
	hintY := h - 1
	if hintY > activeY {
		hint := "1-9 Enter · 0 Clear · Tab ✏️ · Space Pause · Esc Menu"
		if g.pencil {
			hint = "1-9 Mark · 0 Clear · Tab ✒️ · Space Pause · Esc Menu"
		}
		drawCentered(g.screen, w/2, hintY, hint, styleDim)
	}
}

func (g *game) drawHUD() {
	w := g.width
	if w < 1 {
		return
	}
	left := " SUDOKU  " + difficultyLabel[g.difficulty] + "  " + g.modeGlyph()
	clock := formatDuration(g.currentElapsed())
	fillStyle := g.hudStyle()
	for x := 0; x < w; x++ {
		g.screen.SetContent(x, 0, ' ', nil, fillStyle)
	}
	drawText(g.screen, 0, 0, left, fillStyle)
	clockW := runewidth.StringWidth(clock)
	cx := w - clockW - 1
	if cx < 0 {
		cx = 0
	}
	drawText(g.screen, cx, 0, clock, fillStyle)
	start := runewidth.StringWidth(left) + 2
	maxMarks := cx - start
	if maxMarks < 0 {
		maxMarks = 0
	}
	n := g.board.mistakes
	if n > maxMarks {
		n = maxMarks
	}
	for i := 0; i < n; i++ {
		g.screen.SetContent(start+i, 0, mistakeMark, nil, fillStyle)
	}
}

func (g *game) drawBoard(ox, oy int) {
	done := g.board.completedDigits()
	y := oy
	for r := 0; r <= 9; r++ {
		kind := 1
		switch {
		case r == 0:
			kind = 0
		case r == 9:
			kind = 3
		case r == 3 || r == 6:
			kind = 2
		}
		drawHLine(g.screen, ox, y, kind, g.borderStyle())
		y++
		if r < 9 {
			g.drawDigitRow(ox, y, r, done)
			y++
		}
	}
	g.drawCursorBorder(ox, oy)
	g.drawSelectionMarks(ox, oy)
}

func (g *game) drawActiveLine(y int) {
	digits := g.board.activeDigits()
	label := "Active:"
	width := len(label) + 2*len(digits)
	x := (g.width - width) / 2
	if x < 0 {
		x = 0
	}
	drawText(g.screen, x, y, label, styleDefault)
	x += len(label)
	for _, d := range digits {
		drawText(g.screen, x, y, " ", styleDefault)
		x++
		ch := rune(d)
		st := tcell.StyleDefault.Foreground(digitColor[d-'0']).Background(tcell.ColorBlack)
		g.screen.SetContent(x, y, ch, nil, st)
		x++
	}
}

func drawHLine(s tcell.Screen, ox, y, kind int, st tcell.Style) {
	if kind < 0 || kind >= len(hLines) {
		kind = 1
	}
	for x, r := range hLines[kind] {
		s.SetContent(ox+x, y, r, nil, st)
	}
}

var hLines [4][boardCols]rune

func init() {
	for kind := 0; kind < 4; kind++ {
		hLines[kind] = buildHLine(kind)
	}
}

func buildHLine(kind int) [boardCols]rune {
	var left, right, h, tee3, tee9 rune
	switch kind {
	case 0:
		left, right, h, tee3, tee9 = '╔', '╗', '═', '╤', '╦'
	case 3:
		left, right, h, tee3, tee9 = '╚', '╝', '═', '╧', '╩'
	case 2:
		left, right, h, tee3, tee9 = '╠', '╣', '═', '╪', '╬'
	default:
		left, right, h, tee3, tee9 = '╟', '╢', '─', '┼', '╫'
	}
	var line [boardCols]rune
	line[0] = left
	i := 1
	for c := 0; c < 9; c++ {
		line[i] = h
		line[i+1] = h
		line[i+2] = h
		i += 3
		if c < 8 {
			if (c+1)%3 == 0 {
				line[i] = tee9
			} else {
				line[i] = tee3
			}
			i++
		}
	}
	line[boardCols-1] = right
	return line
}

func (g *game) drawDigitRow(ox, y, row int, done [10]bool) {
	s := g.screen
	bs := g.borderStyle()
	s.SetContent(ox, y, '║', nil, bs)
	x := ox + 1
	for c := 0; c < 9; c++ {
		i := row*9 + c
		pad := tcell.StyleDefault.Background(tcell.ColorBlack)
		st := g.cellStyle(i, done)
		ch := rune(g.board.grid[i])
		if ch == '0' {
			if g.board.hasPencil(i) {
				ch = pencilGlyph
				st = g.pencilStyle(i)
				pad = tcell.StyleDefault.Background(tcell.ColorBlack)
			} else {
				ch = ' '
			}
		} else {
			pad = st
		}
		s.SetContent(x, y, ' ', nil, pad)
		s.SetContent(x+1, y, ch, nil, st)
		s.SetContent(x+2, y, ' ', nil, pad)
		x += 3
		if c < 8 {
			sep := '│'
			if (c+1)%3 == 0 {
				sep = '║'
			}
			s.SetContent(x, y, sep, nil, bs)
			x++
		}
	}
	s.SetContent(ox+boardCols-1, y, '║', nil, bs)
}

func (g *game) drawCursorBorder(ox, oy int) {
	if g.flashing() {
		return
	}
	c := g.board.cursor % 9
	r := g.board.cursor / 9
	topY := oy + r*2
	midY := topY + 1
	botY := topY + 2
	leftX := ox + c*4
	rightX := leftX + 4
	s := g.screen
	for x := leftX; x <= rightX; x++ {
		recolor(s, x, topY, g.cursorStyle())
		recolor(s, x, botY, g.cursorStyle())
	}
	recolor(s, leftX, midY, g.cursorStyle())
	recolor(s, rightX, midY, g.cursorStyle())
}

func (g *game) drawSelectionMarks(ox, oy int) {
	c := g.board.cursor % 9
	r := g.board.cursor / 9
	digitX, midY := selectionMarkPos(ox, oy, c, r)
	st := g.titleStyle()
	lw := runeCols(markRowLeft)
	g.putMarker(ox-lw, midY, markRowLeft, st)
	g.putMarker(ox+boardCols, midY, markRowRight, st)
	g.putMarker(digitX, oy-1, markColDown, st)
	g.putMarker(digitX, oy+boardRows, markColUp, st)
}

func selectionMarkPos(ox, oy, col, row int) (digitX, midY int) {
	return ox + col*4 + 2, oy + row*2 + 1
}

func runeCols(ch rune) int {
	w := runewidth.RuneWidth(ch)
	if w < 1 {
		return 1
	}
	return w
}

func (g *game) putMarker(x, y int, ch rune, st tcell.Style) {
	if y < 0 || y >= g.height {
		return
	}
	w := runeCols(ch)
	if x < 0 || x+w > g.width {
		return
	}
	g.screen.SetContent(x, y, ch, nil, st)
}

func recolor(s tcell.Screen, x, y int, st tcell.Style) {
	mainc, combc, _, _ := s.GetContent(x, y)
	s.SetContent(x, y, mainc, combc, st)
}

func (g *game) cellStyle(i int, done [10]bool) tcell.Style {
	st := tcell.StyleDefault.Background(tcell.ColorBlack)
	if g.board.isWrong(i) {
		st = st.Background(tcell.ColorMaroon)
	}
	v := g.board.grid[i]
	if v >= '1' && v <= '9' {
		d := v - '0'
		fg := digitColor[d]
		if done[d] {
			fg = digitCompleteColor
		}
		st = st.Foreground(fg)
		if g.board.isLocked(i) {
			st = st.Bold(true)
		}
	} else {
		st = st.Foreground(tcell.ColorSilver)
	}
	return st
}

func (g *game) modeGlyph() string {
	if g.pencil {
		return modePencil
	}
	return modePen
}

func (g *game) pencilStyle(i int) tcell.Style {
	top := g.board.pencil[i][0]
	bot := g.board.pencil[i][1]
	fg := tcell.ColorBlack
	bg := tcell.ColorBlack
	if top >= '1' && top <= '9' {
		fg = digitColor[top-'0']
	}
	if bot >= '1' && bot <= '9' {
		bg = digitColor[bot-'0']
	}
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

func digitPaint(d byte, done [10]bool) tcell.Color {
	if done[d] {
		return digitCompleteColor
	}
	return digitColor[d]
}

func (g *game) drawPause() {
	w, h := g.width, g.height
	g.screen.Fill(' ', styleDefault)
	g.drawSudokuBanner(1)
	cy := h / 2
	drawCentered(g.screen, w/2, cy-2, "PAUSED", g.titleStyle())
	drawCentered(g.screen, w/2, cy, formatDuration(g.currentElapsed()), styleDefault)
	drawCentered(g.screen, w/2, cy+2, "Space to resume  ·  Esc menu", styleDim)
}

func (g *game) drawConfirm(title, sub, left, right string, leftDestructive bool) {
	w, h := g.width, g.height
	boxW := 48
	boxH := 8
	x := (w - boxW) / 2
	y := (h - boxH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	ov := g.overlayStyle()
	fillRect(g.screen, x, y, boxW, boxH, ov)
	drawCentered(g.screen, w/2, y+1, title, g.overlayTitleStyle())
	drawCentered(g.screen, w/2, y+3, sub, ov)
	g.drawChoiceButtons(x, y+5, boxW, left, right, leftDestructive)
	drawCentered(g.screen, w/2, y+6, "← →  Enter  Esc", ov)
}

func (g *game) drawChoiceButtons(x, y, boxW int, left, right string, leftDestructive bool) {
	leftLabel := "  " + left + "  "
	rightLabel := "  " + right + "  "
	ov := g.overlayStyle()
	leftSt, rightSt := ov, ov
	if g.confirmIndex == 0 {
		if leftDestructive {
			leftSt = g.dangerStyle()
		} else {
			leftSt = g.selectStyle()
		}
	} else {
		if leftDestructive {
			rightSt = g.selectStyle()
		} else {
			rightSt = g.dangerStyle()
		}
	}
	gap := 6
	total := len([]rune(leftLabel)) + gap + len([]rune(rightLabel))
	start := x + (boxW-total)/2
	if start < x+1 {
		start = x + 1
	}
	drawText(g.screen, start, y, leftLabel, leftSt)
	drawText(g.screen, start+len([]rune(leftLabel))+gap, y, rightLabel, rightSt)
}

func (g *game) drawSolved() {
	w, h := g.width, g.height
	boxW, boxH := 44, 9
	x := (w - boxW) / 2
	y := (h - boxH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	ov := g.overlayStyle()
	fillRect(g.screen, x, y, boxW, boxH, ov)
	drawCentered(g.screen, w/2, y+1, "Solved!", g.overlayTitleStyle())
	drawCentered(g.screen, w/2, y+3, "Time: "+formatDuration(time.Duration(g.solvedMs)*time.Millisecond), ov)
	drawCentered(g.screen, w/2, y+4, fmt.Sprintf("Incorrect entries: %d", g.solvedMistakes), ov)
	drawCentered(g.screen, w/2, y+6, "Press Enter", ov)
}

func drawText(s tcell.Screen, x, y int, text string, st tcell.Style) {
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
}

func formatStatLines(rows [][2]string) []string {
	labelW := 0
	for _, row := range rows {
		if n := runewidth.StringWidth(row[0]); n > labelW {
			labelW = n
		}
	}
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = fmt.Sprintf("%*s:  %s", labelW, row[0], row[1])
	}
	return lines
}

func drawStatBlock(s tcell.Screen, cx, y int, lines []string, st tcell.Style) {
	maxW := 0
	for _, line := range lines {
		if n := runewidth.StringWidth(line); n > maxW {
			maxW = n
		}
	}
	x := cx - maxW/2
	if x < 0 {
		x = 0
	}
	for i, line := range lines {
		drawText(s, x, y+i, line, st)
	}
}

func drawCentered(s tcell.Screen, cx, y int, text string, st tcell.Style) {
	w := runewidth.StringWidth(stripVS(text))
	x := cx - w/2
	if x < 0 {
		x = 0
	}
	drawText(s, x, y, text, st)
}

func stripVS(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r != '\uFE0F' && r != '\uFE0E' {
			out = append(out, r)
		}
	}
	return string(out)
}

func fillRect(s tcell.Screen, x, y, w, h int, st tcell.Style) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.SetContent(x+dx, y+dy, ' ', nil, st)
		}
	}
}
