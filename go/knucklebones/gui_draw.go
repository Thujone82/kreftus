//go:build windows || darwin || (linux && cgo)

package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

type fontKind int

const (
	fontRegular fontKind = iota
	fontBold
	fontMono
)

var fontSources [3]*text.GoTextFaceSource

func initFonts() {
	for i, ttf := range [][]byte{goregular.TTF, gobold.TTF, gomono.TTF} {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
		if err != nil {
			log.Fatal(err)
		}
		fontSources[i] = src
	}
}

// Glyphs the Go fonts lack.
var glyphFix = strings.NewReplacer("⚠", "(offline)", "▏", "|")

func (a *guiApp) face(size float64, f fontKind) *text.GoTextFace {
	return &text.GoTextFace{Source: fontSources[f], Size: size * a.vp.scale}
}

// measure is a string's width in design units.
func measure(s string, size float64, f fontKind) float64 {
	return text.Advance(glyphFix.Replace(s), &text.GoTextFace{Source: fontSources[f], Size: size})
}

func wrapPx(s string, size float64, f fontKind, width float64) []string {
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		cur := ""
		for _, w := range strings.Fields(para) {
			switch {
			case cur == "":
				cur = w
			case measure(cur+" "+w, size, f) <= width:
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

func truncPx(s string, size float64, f fontKind, width float64) string {
	if measure(s, size, f) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && measure(string(r)+"…", size, f) > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

func (a *guiApp) c(c tcell.Color) color.RGBA { return rgba(c) }

func (a *guiApp) cardBg() color.RGBA   { return mix(a.c(a.g.pal.bg), a.c(a.g.pal.fg), 0.06) }
func (a *guiApp) buttonBg() color.RGBA { return mix(a.c(a.g.pal.bg), a.c(a.g.pal.fg), 0.13) }

// ---- primitives ----

func roundRectPath(r rect, rad float64) *vector.Path {
	x, y, w, h, q := float32(r.X), float32(r.Y), float32(r.W), float32(r.H), float32(rad)
	if q*2 > w {
		q = w / 2
	}
	if q*2 > h {
		q = h / 2
	}
	var p vector.Path
	p.MoveTo(x+q, y)
	p.LineTo(x+w-q, y)
	p.ArcTo(x+w, y, x+w, y+q, q)
	p.LineTo(x+w, y+h-q)
	p.ArcTo(x+w, y+h, x+w-q, y+h, q)
	p.LineTo(x+q, y+h)
	p.ArcTo(x, y+h, x, y+h-q, q)
	p.LineTo(x, y+q)
	p.ArcTo(x, y, x+q, y, q)
	p.Close()
	return &p
}

func pathOpts(c color.Color) *vector.DrawPathOptions {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	return op
}

func (a *guiApp) fill(dst *ebiten.Image, r rect, c color.Color, rad float64) {
	vector.FillPath(dst, roundRectPath(a.vp.toScreen(r), rad*a.vp.scale), nil, pathOpts(c))
}

func (a *guiApp) stroke(dst *ebiten.Image, r rect, c color.Color, rad, width float64) {
	so := &vector.StrokeOptions{Width: float32(width * a.vp.scale)}
	vector.StrokePath(dst, roundRectPath(a.vp.toScreen(r), rad*a.vp.scale), so, pathOpts(c))
}

func (a *guiApp) circle(dst *ebiten.Image, x, y, rad float64, c color.Color) {
	s := a.vp.scale
	vector.FillCircle(dst, float32(a.vp.offX+x*s), float32(a.vp.offY+y*s), float32(rad*s), c, true)
}

// triangle points left (dir -1), right (1), down (2), or up (-2).
func (a *guiApp) triangle(dst *ebiten.Image, r rect, dir int, c color.Color) {
	sr := a.vp.toScreen(r)
	x, y, w, h := float32(sr.X), float32(sr.Y), float32(sr.W), float32(sr.H)
	var p vector.Path
	switch dir {
	case -1:
		p.MoveTo(x+w, y)
		p.LineTo(x, y+h/2)
		p.LineTo(x+w, y+h)
	case 1:
		p.MoveTo(x, y)
		p.LineTo(x+w, y+h/2)
		p.LineTo(x, y+h)
	case 2:
		p.MoveTo(x, y)
		p.LineTo(x+w, y)
		p.LineTo(x+w/2, y+h)
	default:
		p.MoveTo(x, y+h)
		p.LineTo(x+w, y+h)
		p.LineTo(x+w/2, y)
	}
	p.Close()
	vector.FillPath(dst, &p, nil, pathOpts(c))
}

// text draws s with its vertical center at y.
func (a *guiApp) text(dst *ebiten.Image, s string, x, y, size float64, f fontKind, c color.Color, align text.Align) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(a.vp.offX+x*a.vp.scale, a.vp.offY+y*a.vp.scale)
	op.ColorScale.ScaleWithColor(c)
	op.PrimaryAlign = align
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, glyphFix.Replace(s), a.face(size, f), op)
}

func (a *guiApp) center(dst *ebiten.Image, s string, y, size float64, f fontKind, c color.Color) {
	a.text(dst, s, baseW/2, y, size, f, c, text.AlignCenter)
}

func (a *guiApp) image(dst *ebiten.Image, img *ebiten.Image, r rect) {
	sr := a.vp.toScreen(r)
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(sr.W/float64(b.Dx()), sr.H/float64(b.Dy()))
	op.GeoM.Translate(sr.X, sr.Y)
	dst.DrawImage(img, op)
}

func (a *guiApp) die(dst *ebiten.Image, r rect, v int, bg, pip tcell.Color) {
	rad := r.W * 0.18
	a.fill(dst, rect{r.X + 2, r.Y + 4, r.W, r.H}, color.RGBA{0, 0, 0, 70}, rad)
	a.fill(dst, r, a.c(bg), rad)
	if v < 1 || v > 6 {
		return
	}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if pipRows[v][row][col] == 'o' {
				a.circle(dst, r.X+r.W*(0.25+0.25*float64(col)), r.Y+r.H*(0.25+0.25*float64(row)), r.W*0.085, a.c(pip))
			}
		}
	}
}

func (a *guiApp) button(dst *ebiten.Image, r rect, label string, selected bool, selBg tcell.Color) {
	p := a.g.pal
	if selected {
		a.fill(dst, r, a.c(selBg), 9)
		a.text(dst, label, r.cx(), r.cy(), 17, fontBold, a.c(p.bg), text.AlignCenter)
		return
	}
	a.fill(dst, r, a.buttonBg(), 9)
	hover := r.has(a.mouseX, a.mouseY)
	if hover {
		a.stroke(dst, r, a.c(selBg), 9, 2)
	}
	a.text(dst, label, r.cx(), r.cy(), 17, fontRegular, a.c(p.fg), text.AlignCenter)
}

func (a *guiApp) card(dst *ebiten.Image, r rect, title string) {
	a.fill(dst, rect{r.X + 4, r.Y + 6, r.W, r.H}, color.RGBA{0, 0, 0, 80}, 16)
	a.fill(dst, r, a.cardBg(), 16)
	a.stroke(dst, r, a.c(a.g.pal.frame), 16, 1.5)
	if title != "" {
		a.text(dst, title, r.cx(), r.Y+34, 22, fontBold, a.c(a.g.pal.accent), text.AlignCenter)
	}
}

func (a *guiApp) scrim(dst *ebiten.Image) {
	bg := a.c(a.g.pal.bg)
	vector.FillRect(dst, 0, 0, float32(a.w), float32(a.h), color.NRGBA{bg.R, bg.G, bg.B, 190}, false)
}

func (a *guiApp) footer(dst *ebiten.Image, hint string) {
	a.center(dst, hint, baseH-20, 14, fontRegular, a.c(a.g.pal.dim))
}

func (a *guiApp) smallTitle(dst *ebiten.Image, y float64) {
	a.center(dst, "K N U C K L E B O N E S", y, 26, fontBold, a.c(a.g.pal.title))
}

// ---- screens ----

func (a *guiApp) Draw(dst *ebiten.Image) {
	g := a.g
	dst.Fill(a.c(g.pal.bg))
	switch g.view {
	case viewMenu:
		a.drawMenu(dst)
	case viewSettings:
		a.drawSettings(dst)
	case viewName:
		a.drawPrompt(dst, "Your name", "Shown to your opponent and on your Top 10.", "Save")
	case viewJoin:
		a.drawPrompt(dst, "Join an online game", "Host address (host:port)", "Join")
	case viewLobby:
		a.drawLobby(dst)
	case viewPlay:
		a.drawPlay(dst)
	case viewGuide:
		a.drawGuide(dst)
	case viewStats:
		a.drawStats(dst)
	}
}

func (a *guiApp) drawMenu(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.image(dst, a.logo, menuLogoRect())
	a.center(dst, "K N U C K L E B O N E S", 152, 34, fontBold, a.c(p.title))
	a.center(dst, "The dice game from Cult of the Lamb", 184, 17, fontRegular, a.c(p.dim))
	for i, l := range g.menuLabels() {
		a.button(dst, menuItemRect(i), l, i == g.menuIndex, p.accent)
	}
	a.center(dst, creditLine1, 652, 14, fontRegular, a.c(p.dim))
	a.center(dst, creditLine2, 672, 14, fontRegular, a.c(p.dim))
	a.footer(dst, "Click or ↑↓ to select · Enter · T theme · Esc quit")
}

func caret() string {
	if time.Now().UnixMilli()/500%2 == 0 {
		return "|"
	}
	return ""
}

func (a *guiApp) drawSettings(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.smallTitle(dst, 42)
	a.center(dst, setupTitles[g.setupMode], 86, 24, fontBold, a.c(p.fg))
	a.center(dst, "Game settings", 118, 16, fontRegular, a.c(p.dim))
	rows := g.settingRows()
	for i, row := range rows {
		sel := i == g.setIndex
		if row == rowStart {
			a.button(dst, settingsStartRect(i), "Start game", sel, p.accent)
			continue
		}
		r := settingsRowRect(i)
		a.fill(dst, r, a.cardBg(), 10)
		if sel {
			a.stroke(dst, r, a.c(p.accent), 10, 2)
		}
		label, value := g.settingText(row)
		lc := a.c(p.dim)
		if sel {
			lc = a.c(p.accent)
		}
		a.text(dst, label, r.X+20, r.cy(), 17, fontRegular, lc, text.AlignStart)
		left, right := settingsArrows(r)
		vc := a.c(p.fg)
		if row == rowPlayer2 || row == rowPort {
			field := rect{left.X, r.Y + 6, right.X + right.W - left.X, r.H - 12}
			a.fill(dst, field, a.c(p.emptyBg), 6)
			if sel {
				value += caret()
			}
			a.text(dst, value, field.X+12, field.cy(), 17, fontRegular, vc, text.AlignStart)
			continue
		}
		ac := a.c(p.dim)
		if sel {
			ac, vc = a.c(p.accent), a.c(p.accent)
		}
		a.triangle(dst, rect{left.X + 10, left.Y + 9, 14, 16}, -1, ac)
		a.triangle(dst, rect{right.X + 10, right.Y + 9, 14, 16}, 1, ac)
		f := fontRegular
		if sel {
			f = fontBold
		}
		a.text(dst, value, (left.X+left.W+right.X)/2, r.cy(), 17, f, vc, text.AlignCenter)
	}
	y := settingsRowRect(len(rows)).Y + 20
	if g.setupMode == modeAI {
		a.center(dst, difficultyHints[g.save.Settings.Difficulty], y, 16, fontRegular, a.c(p.dim))
	}
	for i, l := range wrapPx(g.note, 16, fontRegular, 900) {
		a.center(dst, l, y+30+float64(i)*24, 16, fontRegular, a.c(p.lose))
	}
	a.button(dst, backButton(), "Back", false, p.accent)
	a.footer(dst, "Click a setting or ↑↓ select · ←→ change · Enter · Esc back")
}

func (a *guiApp) drawPrompt(dst *ebiten.Image, title, label, okLabel string) {
	g, p := a.g, a.g.pal
	a.image(dst, a.logo, centered(baseW/2, 60, 96, 96))
	a.smallTitle(dst, 186)
	c := promptCard()
	a.card(dst, c, title)
	a.text(dst, label, c.X+30, c.Y+78, 16, fontRegular, a.c(p.dim), text.AlignStart)
	f := promptField()
	a.fill(dst, f, a.c(p.emptyBg), 8)
	a.stroke(dst, f, a.c(p.accent), 8, 1.5)
	shown := g.input
	for measure(shown+"|", 18, fontRegular) > f.W-24 && shown != "" {
		shown = string([]rune(shown)[1:])
	}
	a.text(dst, shown+caret(), f.X+12, f.cy(), 18, fontRegular, a.c(p.fg), text.AlignStart)
	back, ok := promptButtons()
	a.button(dst, back, "Back", false, p.accent)
	a.button(dst, ok, okLabel, true, p.accent)
	if g.note != "" {
		a.center(dst, g.note, c.Y+c.H+30, 16, fontRegular, a.c(p.lose))
	}
	a.footer(dst, "Type · Enter confirm · Esc back")
}

func (a *guiApp) drawLobby(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.image(dst, a.logo, centered(baseW/2, 20, 84, 84))
	a.smallTitle(dst, 124)
	l, _ := g.ctl.(lobby)
	if l == nil {
		return
	}
	c := lobbyCard()
	if port := l.hostPort(); port > 0 {
		a.card(dst, c, "Waiting for an opponent")
		y := c.Y + 78
		a.center(dst, "Share one of these addresses with your opponent:", y, 16, fontRegular, a.c(p.dim))
		y += 40
		for _, addr := range localAddrs() {
			if y > c.Y+c.H-170 {
				break
			}
			a.center(dst, fmt.Sprintf("%s:%d", addr, port), y, 20, fontBold, a.c(p.accent))
			y += 30
		}
		a.center(dst, fmt.Sprintf("localhost:%d (this computer)", port), y, 16, fontRegular, a.c(p.dim))
		y += 44
		a.center(dst, "Waiting time: "+formatWait(time.Since(l.waitingSince())), y, 17, fontRegular, a.c(p.fg))
		st := g.save.Settings
		a.center(dst, roundsLabel(st.BestOf)+" · "+rulesSummary(st.Rules), y+28, 16, fontRegular, a.c(p.dim))
		back, _ := lobbyButtons(false)
		a.button(dst, back, "Cancel", false, p.accent)
		a.footer(dst, "Esc cancel")
		return
	}
	a.card(dst, c, "Join an online game")
	if e := l.lobbyError(); e != "" {
		y := c.Y + 100
		for _, line := range wrapPx(lobbyErrorText(e), 18, fontRegular, c.W-80) {
			a.center(dst, line, y, 18, fontRegular, a.c(p.lose))
			y += 28
		}
		a.center(dst, g.save.Settings.JoinAddr, y+20, 17, fontRegular, a.c(p.dim))
		back, again := lobbyButtons(true)
		a.button(dst, back, "Back", false, p.accent)
		a.button(dst, again, "Retry", true, p.accent)
		a.footer(dst, "Enter retry · Esc back")
		return
	}
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/400%4))
	a.center(dst, "Waiting for game to start"+dots, c.Y+120, 20, fontRegular, a.c(p.fg))
	a.center(dst, g.save.Settings.JoinAddr, c.Y+160, 18, fontBold, a.c(p.accent))
	back, _ := lobbyButtons(false)
	a.button(dst, back, "Cancel", false, p.accent)
	a.footer(dst, "Esc cancel")
}

// ---- play screen ----

func (a *guiApp) drawGrid(dst *ebiten.Image, grid *Grid, isTop bool, m gridMarks, ghostVal int) {
	p := a.g.pal
	for c := 0; c < gridCols; c++ {
		for k := 0; k < gridRows; k++ {
			r := playCell(isTop, c, k)
			v := grid[c][k]
			switch {
			case c == m.ghostCol && k == m.ghostSlot:
				a.die(dst, r, ghostVal, p.ghostBg, p.ghostPip)
			case v == 0:
				a.fill(dst, r, a.c(p.emptyBg), r.W*0.18)
			case c == m.doomCol && v == m.doomVal:
				a.die(dst, r, v, p.doomBg, p.doomPip)
			default:
				bg, pip := a.g.dieColors(grid.Matches(c, v))
				if c == m.lastCol && k == m.lastSlot {
					pip = p.lastPip
				}
				a.die(dst, r, v, bg, pip)
			}
		}
	}
}

func (a *guiApp) drawColumnScores(dst *ebiten.Image, y float64, grid, preview *Grid, changed tcell.Color) {
	for c := 0; c < gridCols; c++ {
		r := playCell(true, c, 0)
		now := grid.ColumnScore(c)
		t, col := fmt.Sprint(now), a.c(a.g.pal.fg)
		if preview != nil {
			if next := preview.ColumnScore(c); next != now {
				t, col = fmt.Sprintf("%d→%d", now, next), a.c(changed)
			}
		}
		a.text(dst, t, r.cx(), y, 20, fontBold, col, text.AlignCenter)
	}
}

func (a *guiApp) drawPlay(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	s := g.ctl.state()
	if s == nil {
		return
	}
	now := time.Now()
	bottom := g.ctl.seat()
	if bottom < 0 {
		bottom = 0
	}
	topP := 1 - bottom
	m := &s.Match

	mover := m.Turn
	var preview *Match
	var pl Placement
	mv, aiming := g.previewMove(s)
	if aiming && m.Legal(mv) {
		pp, next := m.Preview(mv)
		pl, preview = pp, &next
	}

	if aiming {
		col := columnHit(mv.Col)
		top := mover == topP
		r := rect{col.X + 4, guiBotGrid - 6, col.W - 8, guiGridH + 12}
		if top {
			r.Y = guiTopGrid - 6
		}
		hl := a.c(p.accent)
		if !m.Grids[mover].Open(mv.Col) {
			hl = a.c(p.dim)
		}
		a.fill(dst, r, color.NRGBA{hl.R, hl.G, hl.B, 56}, 14)
	}

	for _, pi := range []int{topP, bottom} {
		isTop := pi == topP
		marks := noMarks()
		grid := &m.Grids[pi]
		ghostVal := 0
		if s.Last != nil && s.Last.Player == pi && s.Phase == phasePlay {
			marks.lastCol, marks.lastSlot = s.Last.Col, s.Last.Slot
			if marks.lastSlot >= 0 && grid[marks.lastCol][marks.lastSlot] != s.Last.Die {
				marks.lastCol = -1
			}
		}
		var scorePreview *Grid
		var scoreColor tcell.Color
		switch {
		case preview != nil && pi == mover:
			marks.ghostCol, marks.ghostSlot = pl.Col, pl.Slot
			ghostVal = pl.Die
			scorePreview, scoreColor = &preview.Grids[pi], p.win
			a.drawGrid(dst, &preview.Grids[pi], isTop, marks, ghostVal)
		case preview != nil:
			if !m.Rules.NoDestroy {
				marks.doomCol, marks.doomVal = pl.Col, pl.Die
			}
			scorePreview, scoreColor = &preview.Grids[pi], p.lose
			a.drawGrid(dst, grid, isTop, marks, 0)
		default:
			a.drawGrid(dst, grid, isTop, marks, 0)
		}
		y := guiBotScore
		if isTop {
			y = guiTopScore
		}
		a.drawColumnScores(dst, y, grid, scorePreview, scoreColor)
	}

	if aiming {
		r := playCell(true, mv.Col, 0)
		mc := a.c(p.accent)
		if !m.Grids[mover].Open(mv.Col) {
			mc = a.c(p.dim)
		}
		dir := 2
		if mover == topP {
			dir = -2
		}
		ty := guiMidY + 10
		a.triangle(dst, rect{r.cx() - 34, ty, 16, 14}, dir, mc)
		a.triangle(dst, rect{r.cx() + 18, ty, 16, 14}, dir, mc)
		a.text(dst, fmt.Sprint(mv.Col+1), r.cx(), guiMidY+17, 20, fontBold, mc, text.AlignCenter)
	}

	a.drawHeader(dst, s, now)
	a.drawPlayerPanel(dst, s, topP, true, pl, preview != nil, now)
	a.drawPlayerPanel(dst, s, bottom, false, pl, preview != nil, now)
	a.drawSidePanel(dst, s)

	acts := g.playActions(s)
	for i, b := range playButtons(len(acts)) {
		a.button(dst, b, acts[i].label, false, p.accent)
	}
	hint := "Click a column or ←→ · 1-3/Enter place · L log · H history · R resign · Esc leave"
	if m.Rules.TwoDice {
		hint = "Click a column or ←→ · click a die or Tab to switch · Enter place · L · H · R · Esc"
	}
	if s.Phase != phasePlay {
		hint = "Click an option or ←→ · Enter · V view board · L log · H history · Esc leave"
	}
	a.footer(dst, hint)

	switch {
	case g.overlay == overlayLog:
		a.drawLogOverlay(dst, s)
	case g.overlay == overlayHistory:
		a.drawHistoryOverlay(dst, s)
	case g.overlay == overlayResign:
		a.drawConfirm(dst, "Resign this match?", g.resignWarning(s), "Cancel", "Confirm resignation")
	case g.overlay == overlayLeave:
		a.drawConfirm(dst, "Leave this game?", leaveWarning(s), "Stay", "Leave")
	case s.Phase != phasePlay && !g.hideResult:
		a.drawResult(dst, s)
	}
}

func (a *guiApp) drawHeader(dst *ebiten.Image, s *session, now time.Time) {
	p := a.g.pal
	a.image(dst, a.logo, rect{18, 10, 40, 40})
	a.text(dst, "KNUCKLEBONES", 66, 30, 20, fontBold, a.c(p.title), text.AlignStart)
	t, c := a.g.headerText(s, now)
	a.center(dst, t, 30, 22, fontBold, a.c(c))
	a.text(dst, fmt.Sprintf("Round %d · %s", s.Round, s.roundsLabel()), baseW-24, 30, 16, fontRegular, a.c(p.dim), text.AlignEnd)
}

func (a *guiApp) drawPlayerPanel(dst *ebiten.Image, s *session, pi int, isTop bool, pl Placement, hasPreview bool, now time.Time) {
	g, p := a.g, a.g.pal
	m := &s.Match
	turn := s.Phase == phasePlay && m.Turn == pi
	x, w := guiPanelX, guiPanelW
	y := panelNameY(isTop)

	nc := a.c(p.fg)
	if turn {
		nc = a.c(p.accent)
	}
	a.text(dst, truncPx(g.panelName(s, pi), 22, fontBold, w), x, y, 22, fontBold, nc, text.AlignStart)
	a.text(dst, "Total", x, y+28, 17, fontRegular, a.c(p.dim), text.AlignStart)
	a.text(dst, fmt.Sprint(m.Grids[pi].Score()), x+52, y+28, 20, fontBold, a.c(p.fg), text.AlignStart)
	if info := g.winsText(s, pi); info != "" {
		a.text(dst, info, x, y+52, 16, fontRegular, a.c(p.dim), text.AlignStart)
	}
	if s.TurnLimit > 0 && s.Timeouts[pi] > 0 {
		a.text(dst, fmt.Sprintf("Timeouts: %d / %d", s.Timeouts[pi], maxTimeouts), x, y+74, 16, fontRegular, a.c(p.lose), text.AlignStart)
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
		r := trayDieRect(isTop, i)
		if rolling {
			a.die(dst, r, rollingFace(i), p.dieBg, p.pip)
			continue
		}
		bg, pip := p.dieBg, p.pip
		if m.Dice[1] != 0 && i != selDie {
			bg, pip = p.ghostBg, p.ghostPip
		}
		if m.Dice[1] != 0 && i == selDie {
			a.stroke(dst, rect{r.X - 4, r.Y - 4, r.W + 8, r.H + 8}, a.c(p.accent), 14, 2.5)
		}
		a.die(dst, r, v, bg, pip)
	}
	ty := trayDieRect(isTop, 0).Y + guiTrayDie + 18
	switch {
	case hasPreview:
		a.text(dst, truncPx(gainText(pl), 17, fontRegular, w), x, ty, 17, fontRegular, a.c(p.accent), text.AlignStart)
	case !g.myTurn(s) && !rolling:
		a.text(dst, "Thinking…", x, ty, 17, fontRegular, a.c(p.dim), text.AlignStart)
	}
	if left := g.ctl.turnLeft(now); left > 0 {
		secs := int(left.Seconds() + 0.999)
		c := a.c(p.fg)
		if secs <= 5 {
			c = a.c(p.lose)
		}
		a.text(dst, fmt.Sprintf("Time left: %ds", secs), x, ty+24, 17, fontRegular, c, text.AlignStart)
	}
}

func (a *guiApp) drawSidePanel(dst *ebiten.Image, s *session) {
	g, p := a.g, a.g.pal
	x, w := guiSideX, guiSideW
	y := guiTopGrid + 10
	if s.BestOf != 1 || s.FreePlay || len(s.History) > 0 {
		a.text(dst, truncPx(fmt.Sprintf("Series %d – %d", s.Wins[0], s.Wins[1]), 18, fontBold, w), x, y, 18, fontBold, a.c(p.fg), text.AlignStart)
		a.text(dst, truncPx(s.Names[0]+" v "+s.Names[1], 15, fontRegular, w), x, y+24, 15, fontRegular, a.c(p.dim), text.AlignStart)
		y += 58
	}
	if s.Rules != (Rules{}) {
		for _, line := range wrapPx(rulesSummary(s.Rules), 15, fontRegular, w) {
			a.text(dst, line, x, y, 15, fontRegular, a.c(p.dim), text.AlignStart)
			y += 21
		}
		y += 12
	}
	a.text(dst, "Log", x, y, 18, fontBold, a.c(p.accent), text.AlignStart)
	y += 28
	for i := len(s.Log) - 1; i >= 0 && y < guiBotScore; i-- {
		c := a.c(p.dim)
		if i == len(s.Log)-1 {
			c = a.c(p.fg)
		}
		for j, line := range wrapPx(g.logText(s, s.Log[i]), 15, fontRegular, w-16) {
			if y >= guiBotScore {
				break
			}
			if j == 0 {
				a.circle(dst, x+4, y, 2.5, c)
			}
			a.text(dst, line, x+14, y, 15, fontRegular, c, text.AlignStart)
			y += 21
		}
	}
}

func (a *guiApp) resultLines(s *session) int {
	msg, _ := a.g.resultText(s)
	return len(wrapPx(msg, 22, fontBold, 640))
}

func (a *guiApp) drawResult(dst *ebiten.Image, s *session) {
	g, p := a.g, a.g.pal
	msg, mc := g.resultText(s)
	lines := wrapPx(msg, 22, fontBold, 640)
	c := resultCard(len(lines))
	title := "Round over"
	if s.Phase == phaseGameOver {
		title = "Game over"
	}
	a.card(dst, c, title)
	y := c.Y + 78
	for _, line := range lines {
		a.center(dst, line, y, 22, fontBold, a.c(mc))
		y += 30
	}
	sc := s.Match.Scores()
	a.center(dst, fmt.Sprintf("%s %d  –  %d %s", s.Names[0], sc[0], sc[1], s.Names[1]), y+6, 18, fontRegular, a.c(p.fg))
	if s.BestOf != 1 || s.FreePlay {
		a.center(dst, fmt.Sprintf("Rounds won %d – %d · %s", s.Wins[0], s.Wins[1], s.roundsLabel()), y+34, 16, fontRegular, a.c(p.dim))
	}
	if status, sc := g.resultStatus(s); status != "" {
		a.center(dst, status, y+62, 17, fontRegular, a.c(sc))
	}
	choices := g.resultChoices(s)
	for i, b := range resultButtons(c, len(choices)) {
		a.button(dst, b, choiceLabel(choices[i]), i == g.resultIndex, p.accent)
	}
}

func (a *guiApp) confirmLines(s *session) int {
	warn := leaveWarning(s)
	if a.g.overlay == overlayResign {
		warn = a.g.resignWarning(s)
	}
	return len(wrapPx(warn, 18, fontRegular, 520))
}

func (a *guiApp) drawConfirm(dst *ebiten.Image, title, warn, no, yes string) {
	g, p := a.g, a.g.pal
	a.scrim(dst)
	lines := wrapPx(warn, 18, fontRegular, 520)
	c := confirmCard(len(lines))
	a.card(dst, c, title)
	for i, line := range lines {
		a.center(dst, line, c.Y+82+float64(i)*28, 18, fontRegular, a.c(p.fg))
	}
	bs := confirmButtons(c)
	a.button(dst, bs[0], no, g.confirmIndex == 0, p.accent)
	a.button(dst, bs[1], yes, g.confirmIndex == 1, p.lose)
}

func (a *guiApp) drawScrollBox(dst *ebiten.Image, title string, lines []coloredLine, fromBottom bool, hint string) {
	g, p := a.g, a.g.pal
	a.scrim(dst)
	c := overlayCard()
	a.card(dst, c, title)
	room := guiOverRoom
	maxScroll := max(0, len(lines)-room)
	if g.overlayScroll > maxScroll {
		g.overlayScroll = maxScroll
	}
	start := g.overlayScroll
	if fromBottom {
		start = maxScroll - g.overlayScroll
	}
	for i := 0; i < room && start+i < len(lines); i++ {
		l := lines[start+i]
		a.text(dst, l.text, c.X+30, c.Y+80+float64(i)*guiLineH, 16, fontRegular, a.c(l.color), text.AlignStart)
	}
	a.button(dst, overlayClose(), "Close", false, p.accent)
	a.footer(dst, hint)
}

func (a *guiApp) drawLogOverlay(dst *ebiten.Image, s *session) {
	g := a.g
	var lines []coloredLine
	for _, e := range s.Log {
		c := g.logColor(e)
		for _, line := range wrapPx(g.logText(s, e), 16, fontRegular, overlayCard().W-60) {
			lines = append(lines, coloredLine{line, c})
		}
	}
	if len(lines) == 0 {
		lines = []coloredLine{{"Nothing yet.", g.pal.dim}}
	}
	a.drawScrollBox(dst, "Logs", lines, true, "Wheel or ↑↓ to scroll · L or Esc close")
}

func (a *guiApp) drawHistoryOverlay(dst *ebiten.Image, s *session) {
	a.drawScrollBox(dst, "History", a.g.historyLines(s), false, "Wheel or ↑↓ to scroll · H or Esc close")
}

// ---- guide and stats ----

func (a *guiApp) drawGuide(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.smallTitle(dst, 40)
	lines, kinds := g.guideLines(func(t string) []string { return wrapPx(t, 17, fontRegular, guiGuideW-20) })
	room := guiGuideRoom
	maxScroll := max(0, len(lines)-room)
	if g.guideScroll > maxScroll {
		g.guideScroll = maxScroll
	}
	x := baseW/2 - guiGuideW/2
	for i := 0; i < room && g.guideScroll+i < len(lines); i++ {
		j := g.guideScroll + i
		y := guiGuideTop + float64(i)*guiLineH
		switch kinds[j] {
		case 1:
			a.text(dst, lines[j], x, y, 20, fontBold, a.c(p.accent), text.AlignStart)
		case 2:
			a.text(dst, lines[j], x, y, 16, fontMono, a.c(p.fg), text.AlignStart)
		default:
			a.text(dst, lines[j], x, y, 17, fontRegular, a.c(p.fg), text.AlignStart)
		}
	}
	a.button(dst, backButton(), "Back", false, p.accent)
	hint := "Wheel or ↑↓ to scroll · PgUp/PgDn · Esc back"
	if maxScroll > 0 {
		hint = fmt.Sprintf("%s · %d%%", hint, g.guideScroll*100/maxScroll)
	}
	a.footer(dst, hint)
}

func (a *guiApp) drawStats(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.smallTitle(dst, 40)
	a.center(dst, "Stats", 84, 24, fontBold, a.c(p.fg))
	cols := []float64{150, 520, 600, 690, 780, 860, 950}
	row := func(cells [7]string, y float64, c color.Color, f fontKind) {
		a.text(dst, cells[0], cols[0], y, 17, f, c, text.AlignStart)
		for i := 1; i < 7; i++ {
			a.text(dst, cells[i], cols[i], y, 17, f, c, text.AlignEnd)
		}
	}
	y := 130.0
	row(statsHeader, y, a.c(p.dim), fontRegular)
	y += 30
	for _, r := range g.statsRows() {
		c := a.c(p.fg)
		if r.empty {
			c = a.c(p.dim)
		}
		row(r.cells, y, c, fontRegular)
		y += 28
	}
	y += 24
	a.text(dst, "Top 10 final scores", cols[0], y, 20, fontBold, a.c(p.accent), text.AlignStart)
	y += 34
	if len(g.save.HighScores) == 0 {
		a.text(dst, "Finish a game and your score is added here.", cols[0], y, 17, fontRegular, a.c(p.dim), text.AlignStart)
	}
	for i, e := range g.save.HighScores {
		fc := a.c(p.fg)
		a.text(dst, fmt.Sprintf("%d.", i+1), cols[0]+26, y, 17, fontRegular, a.c(p.dim), text.AlignEnd)
		a.text(dst, truncPx(e.Name, 17, fontRegular, 220), cols[0]+40, y, 17, fontBold, fc, text.AlignStart)
		a.text(dst, fmt.Sprint(e.Score), 420, y, 17, fontBold, fc, text.AlignEnd)
		a.text(dst, "vs "+truncPx(e.Opponent, 17, fontRegular, 220), 460, y, 17, fontRegular, fc, text.AlignStart)
		a.text(dst, fmt.Sprint(e.Against), 760, y, 17, fontRegular, fc, text.AlignEnd)
		a.text(dst, e.Date, 950, y, 17, fontRegular, a.c(p.dim), text.AlignEnd)
		y += 27
	}
	a.button(dst, backButton(), "Back", false, p.accent)
	a.footer(dst, "Esc back")
}
