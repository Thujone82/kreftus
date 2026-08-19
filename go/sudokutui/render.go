package main

import (
	"fmt"
	"strings"
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
const boxTL = '╔'
const boxTR = '╗'
const boxBL = '╚'
const boxBR = '╝'
const boxML = '╠'
const boxMR = '╣'
const boxTD = '╦'
const boxBU = '╩'
const boxH = '═'
const boxV = '║'
const replayInnerW = 9

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

	visible := g.menuItems()
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

	if g.menuIndex == menuReplay {
		g.drawMenuReplayBox(w/2, y)
	} else {
		statRows := [][2]string{}
		if showDetail {
			statRows = [][2]string{
				{"Perfect", fmt.Sprintf("%d", st.Perfect)},
				{"Successes", fmt.Sprintf("%d", st.Successes)},
				{"Error Rate", st.errorRate()},
				{"Failed", fmt.Sprintf("%d", st.Failed)},
				{st.bestLabel(), st.bestValue()},
				{"Average", g.save.averageCompletion(g.difficulty)},
			}
		}
		total := len(puzzlesFor(g.difficulty))
		statRows = append(statRows, [2]string{"Remaining", fmt.Sprintf("%d / %d", remain, total)})
		g.drawStatsBox(w/2, y, "Stats", formatStatLines(statRows))
	}

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
		hint := "1-9 Enter · 0 Clear · Tab/Shift ✏️ · Space Pause · Esc Menu"
		if g.pencilActive() {
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
	return cellStyle(&g.board, i, done)
}

func cellStyle(b *board, i int, done [10]bool) tcell.Style {
	st := tcell.StyleDefault.Background(tcell.ColorBlack)
	if b.isWrong(i) {
		st = st.Background(tcell.ColorMaroon)
	}
	v := b.grid[i]
	if v >= '1' && v <= '9' {
		d := v - '0'
		fg := digitColor[d]
		if done[d] {
			fg = digitCompleteColor
		}
		st = st.Foreground(fg)
		if b.isLocked(i) {
			st = st.Bold(true)
		}
	} else {
		st = st.Foreground(tcell.ColorSilver)
	}
	return st
}

func (g *game) modeGlyph() string {
	if g.pencilActive() {
		return modePencil
	}
	return modePen
}

func (g *game) pencilStyle(i int) tcell.Style {
	return pencilStyle(&g.board, i)
}

func pencilStyle(b *board, i int) tcell.Style {
	top := b.pencil[i][0]
	bot := b.pencil[i][1]
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

func digitHueShift(d byte, shift int) tcell.Color {
	if d < '1' || d > '9' {
		return tcell.ColorWhite
	}
	i := (int(d-'1') + shift) % 9
	if i < 0 {
		i += 9
	}
	return digitColor[i+1]
}

func (g *game) drawPause() {
	w, h := g.width, g.height
	g.screen.Fill(' ', styleDefault)
	g.drawSudokuBanner(1)
	cx := w / 2
	cy := h / 2

	detailY := 4
	drawCentered(g.screen, cx, detailY, strings.ToUpper(difficultyLabel[g.difficulty]), g.titleStyle())
	filled, total := g.board.fillProgress()
	g.drawPauseBar(cx, detailY+2, filled, total)
	if extra := pauseExtraLine(g.board.pencilMarkCount(), g.board.mistakes); extra != "" {
		drawCentered(g.screen, cx, detailY+3, extra, styleDefault)
	}

	pausedY := cy - 2
	if pausedY < detailY+6 {
		pausedY = detailY + 6
	}
	drawCentered(g.screen, cx, pausedY, "PAUSED", g.titleStyle())
	drawCentered(g.screen, cx, pausedY+2, formatDuration(g.currentElapsed()), styleDefault)
	drawCentered(g.screen, cx, pausedY+4, "Space to resume  ·  Esc menu", styleDim)
}

const pauseBarWidth = 20

func (g *game) drawPauseBar(cx, y, filled, total int) {
	pct, n := pauseBarFill(filled, total, pauseBarWidth)
	label := fmt.Sprintf("  %d%%", pct)
	width := pauseBarWidth + len(label)
	x := cx - width/2
	if x < 0 {
		x = 0
	}
	fillSt := tcell.StyleDefault.Foreground(g.accent()).Background(tcell.ColorBlack)
	for i := 0; i < pauseBarWidth; i++ {
		ch := '░'
		st := styleDim
		if i < n {
			ch = '█'
			st = fillSt
		}
		g.screen.SetContent(x+i, y, ch, nil, st)
	}
	drawText(g.screen, x+pauseBarWidth, y, label, styleDefault)
}

func pauseBarFill(filled, total, barW int) (pct, cells int) {
	if barW < 1 {
		return 0, 0
	}
	if total <= 0 {
		return 100, barW
	}
	if filled <= 0 {
		return 0, 0
	}
	if filled >= total {
		return 100, barW
	}
	pct = filled * 100 / total
	cells = filled * barW / total
	if cells < 1 {
		cells = 1
	}
	return pct, cells
}

func pauseExtraLine(pencils, errors int) string {
	switch {
	case pencils > 0 && errors > 0:
		return fmt.Sprintf("Pencil Marks: %d  Errors: %d", pencils, errors)
	case pencils > 0:
		return fmt.Sprintf("Pencil Marks: %d", pencils)
	case errors > 0:
		return fmt.Sprintf("Errors: %d", errors)
	default:
		return ""
	}
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
	g.drawSudokuBanner(1)

	elapsed := formatDuration(time.Duration(g.solvedMs) * time.Millisecond)
	stats, badge := solvedBody(elapsed, g.solvedMistakes)
	record := newCompletionBanner(g.solvedNewRecord, g.save.statsFor(g.difficulty).bestLabel())
	title := solvedTitle(g.difficulty)

	innerW := 28
	for _, line := range stats {
		if n := runewidth.StringWidth(line); n+4 > innerW {
			innerW = n + 4
		}
	}
	for _, line := range []string{badge, record, title} {
		if n := runewidth.StringWidth(line); n+4 > innerW {
			innerW = n + 4
		}
	}
	boxW := innerW + 2
	frameH := 11
	totalW := boxW + replayInnerW + 1
	x := (w - totalW) / 2
	y := (h - frameH) / 2
	if x < 0 {
		x = 0
	}
	if y < 3 {
		y = 3
	}

	border := tcell.StyleDefault.Foreground(g.accent()).Background(tcell.ColorBlack).Bold(true)
	titleFill := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(g.accent2()).Bold(true)
	titleFg := tcell.StyleDefault.Foreground(g.accent3()).Background(g.accent2()).Bold(true)
	perfectSt := tcell.StyleDefault.Foreground(g.accent2()).Background(tcell.ColorBlack).Bold(true)
	hintSt := tcell.StyleDefault.Foreground(g.accent3()).Background(tcell.ColorBlack)

	g.drawSolvedStatsBox(x, y, boxW, innerW, frameH, title, stats, badge, record, border, titleFill, titleFg, perfectSt, hintSt)
	g.drawSolvedReplayPane(x+boxW-1, y, frameH, border)
}

func (g *game) drawSolvedStatsBox(x, y, boxW, innerW, frameH int, title string, stats []string, badge, record string, border, titleFill, titleFg, perfectSt, hintSt tcell.Style) {
	drawBoxRow(g.screen, x, y, boxW, boxTL, boxH, boxTD, border)
	fillRect(g.screen, x+1, y+1, innerW, 1, titleFill)
	g.screen.SetContent(x, y+1, boxV, nil, border)
	g.screen.SetContent(x+boxW-1, y+1, boxV, nil, border)
	drawCentered(g.screen, x+boxW/2, y+1, title, titleFg)
	drawBoxRow(g.screen, x, y+2, boxW, boxML, boxH, boxMR, border)

	for dy := 3; dy <= 7; dy++ {
		g.screen.SetContent(x, y+dy, boxV, nil, border)
		g.screen.SetContent(x+boxW-1, y+dy, boxV, nil, border)
	}
	drawStatBlock(g.screen, x+boxW/2, y+4, stats, styleDefault)
	if badge != "" {
		drawCentered(g.screen, x+boxW/2, y+5, badge, perfectSt)
	}
	if record != "" {
		drawCentered(g.screen, x+boxW/2, y+6, record, perfectSt)
	}

	drawBoxRow(g.screen, x, y+8, boxW, boxML, boxH, boxMR, border)
	g.screen.SetContent(x, y+9, boxV, nil, border)
	g.screen.SetContent(x+boxW-1, y+9, boxV, nil, border)
	drawCentered(g.screen, x+boxW/2, y+9, "Press Enter", hintSt)
	drawBoxRow(g.screen, x, y+frameH-1, boxW, boxBL, boxH, boxBU, border)
}

func (g *game) drawSolvedReplayPane(splitX, y, frameH int, border tcell.Style) {
	innerX := splitX + 1
	rightX := innerX + replayInnerW
	for dx := 0; dx < replayInnerW; dx++ {
		g.screen.SetContent(innerX+dx, y, boxH, nil, border)
		g.screen.SetContent(innerX+dx, y+frameH-1, boxH, nil, border)
	}
	g.screen.SetContent(rightX, y, boxTR, nil, border)
	for dy := 1; dy < frameH-1; dy++ {
		g.screen.SetContent(rightX, y+dy, boxV, nil, border)
	}
	g.screen.SetContent(rightX, y+frameH-1, boxBR, nil, border)
	pb, shift, celebrate := g.playbackView()
	g.drawReplayGrid(innerX, y+1, &pb, shift, celebrate)
}

func (g *game) drawReplayGrid(x, y int, b *board, hueShift int, celebrate bool) {
	done := b.completedDigits()
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			i := r*9 + c
			ch, st := replayCellGlyph(b, i, done, hueShift, celebrate)
			g.screen.SetContent(x+c, y+r, ch, nil, st)
		}
	}
}

func replayCellGlyph(b *board, i int, done [10]bool, hueShift int, celebrate bool) (rune, tcell.Style) {
	if celebrate {
		d := b.solution[i]
		if d < '1' || d > '9' {
			d = b.grid[i]
		}
		st := tcell.StyleDefault.Foreground(digitHueShift(d, hueShift)).Background(tcell.ColorBlack)
		if b.isGiven(i) {
			st = st.Bold(true)
		}
		return rune(d), st
	}
	if emptyCell(b.grid[i]) {
		if b.hasPencil(i) {
			return pencilGlyph, pencilStyle(b, i)
		}
		return ' ', styleDefault
	}
	return rune(b.grid[i]), cellStyle(b, i, done)
}

func solvedTitle(difficulty string) string {
	label := difficultyLabel[difficulty]
	if label == "" {
		label = difficulty
	}
	return label + " Solved!"
}

func solvedBody(elapsed string, mistakes int) (stats []string, badge string) {
	if mistakes == 0 {
		return formatStatLines([][2]string{{"Time", elapsed}}), "Perfect Finish!"
	}
	return formatStatLines([][2]string{
		{"Time", elapsed},
		{"Errors", fmt.Sprintf("%d", mistakes)},
	}), ""
}

func newCompletionBanner(isNew bool, label string) string {
	if !isNew {
		return ""
	}
	return "New " + label + " Completion!"
}

func drawBoxRow(s tcell.Screen, x, y, w int, left, mid, right rune, st tcell.Style) {
	s.SetContent(x, y, left, nil, st)
	s.SetContent(x+w-1, y, right, nil, st)
	for dx := 1; dx < w-1; dx++ {
		s.SetContent(x+dx, y, mid, nil, st)
	}
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

func (st *diffStats) bestLabel() string {
	if st != nil && st.FastestMistakes != nil && *st.FastestMistakes == 0 {
		return "Fastest"
	}
	return "Best"
}

func (st *diffStats) bestValue() string {
	if st == nil || st.FastestMs == nil {
		return "—"
	}
	v := formatDuration(time.Duration(*st.FastestMs) * time.Millisecond)
	if st.FastestMistakes != nil && *st.FastestMistakes > 0 {
		v += fmt.Sprintf("  (%d incorrect)", *st.FastestMistakes)
	}
	return v
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

const (
	statsBoxTL = '┌'
	statsBoxTR = '┐'
	statsBoxBL = '└'
	statsBoxBR = '┘'
	statsBoxH  = '─'
	statsBoxV  = '│'
)

func statsBoxWidth(lines []string, title string) int {
	maxW := runewidth.StringWidth(title) + 2
	for _, line := range lines {
		if n := runewidth.StringWidth(line); n > maxW {
			maxW = n
		}
	}
	return maxW + 4
}

func titledBoxTop(boxW int, title string) string {
	if boxW < 2 {
		return ""
	}
	inner := boxW - 2
	dashes := make([]rune, inner)
	for i := range dashes {
		dashes[i] = statsBoxH
	}
	tw := runewidth.StringWidth(title)
	start := (inner - tw) / 2
	if start < 0 {
		start = 0
	}
	i := 0
	for _, r := range title {
		if start+i >= inner {
			break
		}
		dashes[start+i] = r
		i++
	}
	return string(statsBoxTL) + string(dashes) + string(statsBoxTR)
}

func (g *game) drawStatsBox(cx, y int, title string, lines []string) {
	st := g.titleStyle()
	textSt := styleDefault
	boxW := statsBoxWidth(lines, title)
	x := cx - boxW/2
	if x < 0 {
		x = 0
	}
	top := titledBoxTop(boxW, " "+title+" ")
	drawText(g.screen, x, y, top, st)
	for i, line := range lines {
		row := y + 1 + i
		g.screen.SetContent(x, row, statsBoxV, nil, st)
		g.screen.SetContent(x+boxW-1, row, statsBoxV, nil, st)
		drawText(g.screen, x+2, row, line, textSt)
	}
	bot := y + 1 + len(lines)
	g.screen.SetContent(x, bot, statsBoxBL, nil, st)
	g.screen.SetContent(x+boxW-1, bot, statsBoxBR, nil, st)
	for dx := 1; dx < boxW-1; dx++ {
		g.screen.SetContent(x+dx, bot, statsBoxH, nil, st)
	}
}

func (g *game) drawMenuReplayBox(cx, y int) {
	st := g.titleStyle()
	boxW := replayInnerW + 2
	x := cx - boxW/2
	if x < 0 {
		x = 0
	}
	title := "Replay"
	top := titledBoxTop(boxW, title)
	drawText(g.screen, x, y, top, st)
	tx := x + (boxW-len(title))/2
	drawText(g.screen, tx, y, title, g.selectStyle())
	innerX := x + 1
	for r := 0; r < 9; r++ {
		row := y + 1 + r
		g.screen.SetContent(x, row, statsBoxV, nil, st)
		g.screen.SetContent(x+boxW-1, row, statsBoxV, nil, st)
	}
	pb, shift, celebrate := g.playbackView()
	g.drawReplayGrid(innerX, y+1, &pb, shift, celebrate)
	bot := y + 10
	g.screen.SetContent(x, bot, statsBoxBL, nil, st)
	g.screen.SetContent(x+boxW-1, bot, statsBoxBR, nil, st)
	for dx := 1; dx < boxW-1; dx++ {
		g.screen.SetContent(x+dx, bot, statsBoxH, nil, st)
	}
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
