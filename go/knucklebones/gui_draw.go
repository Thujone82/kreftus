//go:build windows || darwin || (linux && cgo)

package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
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

type faceKey struct {
	sizeBits   uint64
	f          fontKind
	scaleMilli int
}

type measureKey struct {
	s    string
	size float64
	f    fontKind
}

var (
	measureMu    sync.Mutex
	measureFaces [3]map[float64]*text.GoTextFace
	measureCache = make(map[measureKey]float64, 512)
)

func (a *guiApp) face(size float64, f fontKind) *text.GoTextFace {
	scaleMilli := int(math.Round(a.vp.scale * 1000))
	if scaleMilli <= 0 {
		scaleMilli = 1
	}
	if a.faceCache == nil {
		a.faceCache = make(map[faceKey]*text.GoTextFace, 32)
	}
	key := faceKey{sizeBits: math.Float64bits(size), f: f, scaleMilli: scaleMilli}
	if fc := a.faceCache[key]; fc != nil {
		return fc
	}
	if len(a.faceCache) >= 128 {
		clear(a.faceCache)
	}
	fc := &text.GoTextFace{Source: fontSources[f], Size: size * a.vp.scale}
	a.faceCache[key] = fc
	return fc
}

// measure is a string's width in design units.
func measure(s string, size float64, f fontKind) float64 {
	if s == "" {
		return 0
	}
	if fontSources[f] == nil {
		return float64(len([]rune(s))) * size * 0.5
	}
	key := measureKey{s: s, size: size, f: f}
	measureMu.Lock()
	defer measureMu.Unlock()
	if w, ok := measureCache[key]; ok {
		return w
	}
	if measureFaces[f] == nil {
		measureFaces[f] = make(map[float64]*text.GoTextFace, 16)
	}
	fc := measureFaces[f][size]
	if fc == nil {
		fc = &text.GoTextFace{Source: fontSources[f], Size: size}
		measureFaces[f][size] = fc
	}
	w := text.Advance(glyphFix.Replace(s), fc)
	if len(measureCache) >= 2048 {
		clear(measureCache)
	}
	measureCache[key] = w
	return w
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

// caretIndexAt maps a click x (design units) onto a rune caret index.
func caretIndexAt(s string, clickX, textX, size float64, f fontKind) int {
	rel := clickX - textX
	if rel <= 0 {
		return 0
	}
	r := []rune(s)
	idx := sort.Search(len(r)+1, func(i int) bool {
		return measure(string(r[:i]), size, f) >= rel
	})
	if idx > len(r) {
		return len(r)
	}
	if idx > 0 {
		d0 := rel - measure(string(r[:idx-1]), size, f)
		d1 := measure(string(r[:idx]), size, f) - rel
		if d0 <= d1 {
			return idx - 1
		}
	}
	return idx
}

// visibleWithCaret scrolls s so the caret stays inside maxW and returns the
// visible slice plus the caret offset within that slice.
func visibleWithCaret(s string, pos int, maxW, size float64, f fontKind) (shown string, caret int) {
	r := []rune(s)
	pos = clampRunePos(s, pos)
	start := sort.Search(pos, func(i int) bool {
		return measure(string(r[i:pos])+"|", size, f) <= maxW
	})
	hi := sort.Search(len(r)-pos+1, func(k int) bool {
		return measure(string(r[start:pos+k])+"|", size, f) > maxW
	})
	end := pos + hi - 1
	if end < pos {
		hi = sort.Search(pos-start+1, func(k int) bool {
			return measure(string(r[start:start+k])+"|", size, f) > maxW
		})
		end = start + hi - 1
		if end < start {
			end = start
		}
	}
	return string(r[start:end]), pos - start
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

type dieKey struct {
	v    int
	mark int
	bg   tcell.Color
	pip  tcell.Color
	w    int
	h    int
}

func drawDieMark(dst *ebiten.Image, w, h float64, mark int, c color.Color) {
	if mark == markNone {
		return
	}
	sw := float32(math.Max(1.5, w*0.028))
	so := &vector.StrokeOptions{Width: sw, LineCap: vector.LineCapRound}
	var p vector.Path
	switch mark {
	case markPair:
		x1, x2 := float32(w*0.39), float32(w*0.61)
		p.MoveTo(x1, float32(h*0.075))
		p.LineTo(x2, float32(h*0.075))
		p.MoveTo(x1, float32(h*0.128))
		p.LineTo(x2, float32(h*0.128))
	case markTriple:
		x1, x2 := float32(w*0.39), float32(w*0.61)
		y1, y2 := float32(h*0.072), float32(h*0.128)
		p.MoveTo(x1, y1)
		p.LineTo(x2, y1)
		p.MoveTo(x1, y2)
		p.LineTo(x2, y2)
		vx1, vx2 := float32(w*0.45), float32(w*0.55)
		vy1, vy2 := float32(h*0.042), float32(h*0.158)
		p.MoveTo(vx1, vy1)
		p.LineTo(vx1, vy2)
		p.MoveTo(vx2, vy1)
		p.LineTo(vx2, vy2)
	case markDoom:
		x1, x2 := float32(w*0.43), float32(w*0.57)
		y1, y2 := float32(h*0.045), float32(h*0.155)
		p.MoveTo(x1, y1)
		p.LineTo(x2, y2)
		p.MoveTo(x2, y1)
		p.LineTo(x1, y2)
	}
	vector.StrokePath(dst, &p, so, pathOpts(c))
}

func drawDieSprite(dst *ebiten.Image, w, h, scale float64, v, mark int, bg, pip color.Color) {
	rad := w * 0.18
	sh := roundRectPath(rect{2 * scale, 4 * scale, w, h}, rad)
	vector.FillPath(dst, sh, nil, pathOpts(color.RGBA{0, 0, 0, 70}))
	bd := roundRectPath(rect{0, 0, w, h}, rad)
	vector.FillPath(dst, bd, nil, pathOpts(bg))
	if v < 1 || v > 6 {
		return
	}
	pr := float32(w * 0.085)
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if pipRows[v][row][col] == 'o' {
				vector.FillCircle(dst, float32(w*(0.25+0.25*float64(col))), float32(h*(0.25+0.25*float64(row))), pr, pip, true)
			}
		}
	}
	drawDieMark(dst, w, h, mark, pip)
}

func (a *guiApp) die(dst *ebiten.Image, r rect, v int, bg, pip tcell.Color) {
	a.dieMarked(dst, r, v, markNone, bg, pip)
}

func (a *guiApp) dieMarked(dst *ebiten.Image, r rect, v, mark int, bg, pip tcell.Color) {
	sr := a.vp.toScreen(r)
	iw, ih := int(math.Round(sr.W)), int(math.Round(sr.H))
	if iw < 2 || ih < 2 {
		return
	}
	scaleMilli := int(math.Round(a.vp.scale * 1000))
	if a.dieCache == nil || a.dieScaleMilli != scaleMilli {
		a.dieCache = make(map[dieKey]*ebiten.Image, 32)
		a.dieScaleMilli = scaleMilli
	}
	key := dieKey{v: v, mark: mark, bg: bg, pip: pip, w: iw, h: ih}
	img := a.dieCache[key]
	if img == nil {
		if len(a.dieCache) >= 128 {
			clear(a.dieCache)
		}
		padX := int(math.Ceil(3 * a.vp.scale))
		padY := int(math.Ceil(5 * a.vp.scale))
		if padX < 1 {
			padX = 1
		}
		if padY < 1 {
			padY = 1
		}
		img = ebiten.NewImage(iw+padX, ih+padY)
		drawDieSprite(img, float64(iw), float64(ih), a.vp.scale, v, mark, a.c(bg), a.c(pip))
		a.dieCache[key] = img
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(math.Round(sr.X), math.Round(sr.Y))
	dst.DrawImage(img, &op)
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
	now := time.Now()
	sig, anim := a.frameSignature(now)
	b := dst.Bounds()
	dw, dh := b.Dx(), b.Dy()
	if dw <= 0 || dh <= 0 {
		return
	}
	if a.frame == nil || a.frame.Bounds().Dx() != dw || a.frame.Bounds().Dy() != dh {
		a.frame = ebiten.NewImage(dw, dh)
		a.dirty = true
	}
	if a.dirty || anim || sig != a.lastSig {
		a.drawScene(a.frame)
		a.lastSig = sig
		a.dirty = anim
	}
	dst.DrawImage(a.frame, nil)
}

func (a *guiApp) drawScene(dst *ebiten.Image) {
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
		a.drawJoin(dst)
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
	subtitle := "Game settings"
	if g.setupMode == setupOptions {
		subtitle = "Appearance & audio"
	}
	a.center(dst, subtitle, 118, 16, fontRegular, a.c(p.dim))
	rows := g.settingRows()
	for i, row := range rows {
		sel := i == g.setIndex
		label, value := g.settingText(row)
		if row == rowStart || row == rowDone {
			a.button(dst, settingsStartRect(i), label, sel, p.accent)
			continue
		}
		r := settingsRowRect(i)
		a.fill(dst, r, a.cardBg(), 10)
		if sel {
			a.stroke(dst, r, a.c(p.accent), 10, 2)
		}
		lc := a.c(p.dim)
		if sel {
			lc = a.c(p.accent)
		}
		a.text(dst, label, r.X+20, r.cy(), 17, fontRegular, lc, text.AlignStart)
		left, right := settingsArrows(r)
		vc := a.c(p.fg)
		if row == rowPlayer2 || row == rowPort || row == rowRoomCode {
			field := settingsTextField(r)
			a.fill(dst, field, a.c(p.emptyBg), 6)
			if sel {
				edit := value
				switch row {
				case rowPlayer2:
					edit = g.save.Settings.Player2
				case rowPort:
					edit = g.portEditString()
				case rowRoomCode:
					edit = g.hostRoomCode
				}
				shown, cpos := visibleWithCaret(edit, g.inputPos, field.W-24, 17, fontRegular)
				a.drawCaretText(dst, shown, cpos, field.X+12, field.cy(), 17, vc)
			} else {
				tc := vc
				if row == rowRoomCode && g.hostRoomCode == "" {
					tc = a.c(p.dim)
				}
				a.text(dst, value, field.X+12, field.cy(), 17, fontRegular, tc, text.AlignStart)
			}
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
	switch g.setupMode {
	case modeAI:
		a.center(dst, difficultyHints[g.save.Settings.Difficulty], y, 16, fontRegular, a.c(p.dim))
	case setupHost:
		a.center(dst, "Room code is optional and never saved to disk.", y, 16, fontRegular, a.c(p.dim))
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
	shown, cpos := visibleWithCaret(g.input, g.inputPos, f.W-24, 18, fontRegular)
	a.drawCaretText(dst, shown, cpos, f.X+12, f.cy(), 18, a.c(p.fg))
	back, ok := promptButtons()
	a.button(dst, back, "Back", false, p.accent)
	a.button(dst, ok, okLabel, true, p.accent)
	if g.note != "" {
		a.center(dst, g.note, c.Y+c.H+30, 16, fontRegular, a.c(p.lose))
	}
	a.footer(dst, "Type · click or ←→ move caret · Enter confirm · Esc back")
}

func (a *guiApp) drawCaretText(dst *ebiten.Image, s string, pos int, x, y, size float64, col color.Color) {
	r := []rune(s)
	pos = clampRunePos(s, pos)
	before := string(r[:pos])
	after := string(r[pos:])
	a.text(dst, before, x, y, size, fontRegular, col, text.AlignStart)
	cx := x + measure(before, size, fontRegular)
	a.text(dst, after, cx, y, size, fontRegular, col, text.AlignStart)
	if caret() != "" {
		a.text(dst, "|", cx, y, size, fontRegular, col, text.AlignStart)
	}
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
		y := c.Y + 76
		a.center(dst, "Share one of these addresses with your opponent:", y, 16, fontRegular, a.c(p.dim))
		y += 34
		st := g.save.Settings
		for _, addr := range lobbyHostAddresses(st.BindIP, port) {
			if y > c.Y+c.H-190 {
				break
			}
			a.center(dst, addr, y, 20, fontBold, a.c(p.accent))
			y += 28
		}
		if st.BindIP != "127.0.0.1" && y <= c.Y+c.H-160 {
			a.center(dst, fmt.Sprintf("localhost:%d (this computer)", port), y, 16, fontRegular, a.c(p.dim))
			y += 28
		}
		if rc := l.hostRoomCode(); rc != "" && y <= c.Y+c.H-130 {
			a.center(dst, "Room code: "+rc, y, 17, fontBold, a.c(p.accent))
			y += 28
		}
		y += 10
		a.center(dst, "Waiting time: "+formatWait(time.Since(l.waitingSince())), y, 17, fontRegular, a.c(p.fg))
		a.center(dst, roundsLabel(st.BestOf)+" · "+rulesSummary(st.Rules), y+26, 16, fontRegular, a.c(p.dim))
		back, _ := lobbyButtons(false)
		a.button(dst, back, "Cancel", false, p.accent)
		a.footer(dst, "Esc cancel")
		return
	}
	a.card(dst, c, "Join Game")
	if e := l.lobbyError(); e != "" {
		y := c.Y + 100
		for _, line := range wrapPx(lobbyErrorText(e), 18, fontRegular, c.W-80) {
			a.center(dst, line, y, 18, fontRegular, a.c(p.lose))
			y += 28
		}
		a.center(dst, g.save.Settings.JoinAddr, y+20, 17, fontRegular, a.c(p.dim))
		retry := lobbyCanRetry(e)
		back, again := lobbyButtons(retry)
		a.button(dst, back, "Back", !retry || g.lobbyBackFocus, p.accent)
		if retry {
			a.button(dst, again, "Retry", !g.lobbyBackFocus, p.accent)
			a.footer(dst, "←→ select · Enter confirm · Esc back")
		} else {
			a.footer(dst, "Enter / Esc back")
		}
		return
	}
	dots := strings.Repeat(".", int(time.Now().UnixMilli()/400%4))
	a.center(dst, "Waiting for game to start"+dots, c.Y+120, 20, fontRegular, a.c(p.fg))
	a.center(dst, g.save.Settings.JoinAddr, c.Y+160, 18, fontBold, a.c(p.accent))
	back, _ := lobbyButtons(false)
	a.button(dst, back, "Cancel", false, p.accent)
	a.footer(dst, "Esc cancel")
}

func (a *guiApp) drawJoin(dst *ebiten.Image) {
	g, p := a.g, a.g.pal
	a.image(dst, a.logo, centered(baseW/2, 48, 84, 84))
	a.smallTitle(dst, 158)
	c := joinCard()
	a.card(dst, c, "Join Game")
	labels := [2]string{"Host address (host:port)", "Room code (optional, leave blank if open)"}
	vals := [2]string{g.input, g.joinCode}
	placeholders := [2]string{"", "(optional)"}
	for i := 0; i < 2; i++ {
		f := joinFieldRect(i)
		active := g.joinField == i
		lc := a.c(p.dim)
		if active {
			lc = a.c(p.accent)
		}
		a.text(dst, labels[i], f.X, f.Y-14, 15, fontRegular, lc, text.AlignStart)
		a.fill(dst, f, a.c(p.emptyBg), 8)
		if active {
			a.stroke(dst, f, a.c(p.accent), 8, 2)
			shown, cpos := visibleWithCaret(vals[i], g.inputPos, f.W-24, 18, fontRegular)
			a.drawCaretText(dst, shown, cpos, f.X+12, f.cy(), 18, a.c(p.fg))
		} else if vals[i] == "" && placeholders[i] != "" {
			a.text(dst, placeholders[i], f.X+12, f.cy(), 18, fontRegular, a.c(p.dim), text.AlignStart)
		} else {
			a.text(dst, truncPx(vals[i], 18, fontRegular, f.W-24), f.X+12, f.cy(), 18, fontRegular, a.c(p.fg), text.AlignStart)
		}
	}
	if g.note != "" {
		f1 := joinFieldRect(1)
		a.text(dst, truncPx(g.note, 15, fontRegular, c.W-60), c.X+30, f1.Y+f1.H+18, 15, fontRegular, a.c(p.lose), text.AlignStart)
	}
	back, ok := joinButtons()
	a.button(dst, back, "Back", false, p.accent)
	a.button(dst, ok, "Join", true, p.accent)
	a.footer(dst, "Tab / ↑↓ switch field · Enter join · Esc back")
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
				a.dieMarked(dst, r, v, a.g.cellMark(0, true), p.doomBg, p.doomPip)
			default:
				bg, pip := a.g.dieColors(grid.Matches(c, v))
				if c == m.lastCol && k == m.lastSlot {
					pip = p.lastPip
				}
				a.dieMarked(dst, r, v, a.g.cellMark(grid.Matches(c, v), false), bg, pip)
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
	bottom := g.bottomSeat()
	topP := 1 - bottom
	m := &s.Match
	lastPl := s.Last
	replayNote := ""
	var replayMatch Match
	if g.replayActive && len(g.replayFrames) > 0 {
		step := g.replayStep
		if step < 0 {
			step = 0
		}
		if step >= len(g.replayFrames) {
			step = len(g.replayFrames) - 1
		}
		rf := &g.replayFrames[step]
		replayMatch = rf.match
		m = &replayMatch
		lastPl = rf.last
		replayNote = fmt.Sprintf("REPLAY %d/%d · %s", rf.step, rf.total, rf.desc)
	}

	mover := m.Turn
	var preview *Match
	var pl Placement
	var mv Move
	aiming := false
	if !g.replayActive {
		mv, aiming = g.previewMove(s)
		if aiming && m.Legal(mv) {
			pp, next := m.Preview(mv)
			pl, preview = pp, &next
		}
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

	prog := -1.0
	if !g.replayActive {
		prog = effectProgress(g.effectAt, now)
	}

	for _, pi := range []int{topP, bottom} {
		isTop := pi == topP
		gy := guiBotGrid
		if isTop {
			gy = guiTopGrid
		}
		if !g.replayActive && s.Phase == phasePlay && pi == mover {
			ac := a.c(p.accent)
			a.stroke(dst, rect{guiGridX - 8, gy - 8, guiGridW + 16, guiGridH + 16}, color.NRGBA{ac.R, ac.G, ac.B, 150}, 14, 2)
		}
		marks := noMarks()
		grid := &m.Grids[pi]
		ghostVal := 0
		if lastPl != nil && lastPl.Player == pi && (s.Phase == phasePlay || g.replayActive) {
			marks.lastCol, marks.lastSlot = lastPl.Col, lastPl.Slot
			if marks.lastSlot >= 0 && grid[marks.lastCol][marks.lastSlot] != lastPl.Die {
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
		if prog >= 0 {
			isGain := pi == g.effectLast.Player && g.effectLast.Gain > 0
			isLoss := pi != g.effectLast.Player && g.effectLast.Destroyed > 0 && g.effectLast.Lost > 0
			if isGain || isLoss {
				colR := columnHit(g.effectLast.Col)
				if isLoss {
					lc := a.c(p.lose)
					a.fill(dst, rect{colR.X + 4, gy - 4, colR.W - 8, guiGridH + 8}, color.NRGBA{lc.R, lc.G, lc.B, uint8((1 - prog) * 115)}, 12)
				}
				badgeText := fmt.Sprintf("+%d", g.effectLast.Gain)
				bc := a.c(p.win)
				if isLoss {
					badgeText = fmt.Sprintf("-%d", g.effectLast.Lost)
					bc = a.c(p.lose)
				}
				alpha := uint8((1 - prog) * 235)
				by := gy + guiGridH*0.5 - prog*26
				pill := rect{colR.cx() - 28, by - 13, 56, 26}
				bgc := a.cardBg()
				a.fill(dst, pill, color.NRGBA{bgc.R, bgc.G, bgc.B, alpha}, 13)
				a.stroke(dst, pill, color.NRGBA{bc.R, bc.G, bc.B, alpha}, 13, 1.5)
				a.text(dst, badgeText, pill.cx(), pill.cy(), 15, fontBold, color.NRGBA{bc.R, bc.G, bc.B, alpha}, text.AlignCenter)
			}
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
	} else if replayNote != "" {
		a.center(dst, truncPx(replayNote, 16, fontBold, guiGridW+220), guiMidY+17, 16, fontBold, a.c(p.accent))
	}

	a.drawHeader(dst, s, now)
	a.drawPlayerPanel(dst, s, m, topP, true, pl, preview != nil, now)
	a.drawPlayerPanel(dst, s, m, bottom, false, pl, preview != nil, now)
	a.drawSidePanel(dst, s)

	acts := g.playActions(s)
	for i, b := range playButtons(len(acts)) {
		a.button(dst, b, acts[i].label, false, p.accent)
	}
	hint := "Click a column or ←→ · 1-3/Enter place · C marks · P prob · S sound · L · H · R · Esc"
	if m.Rules.TwoDice {
		hint = "Click a column or ←→ · click die or Tab/W · Enter · C · P · S · L · H · R · Esc"
	}
	if g.replayActive {
		hint = "←→ step · Home/End jump · Enter or Esc done"
	} else if s.Phase != phasePlay {
		hint = "Click an option or ←→ · Enter · V board · L log · H history · Esc leave"
		if canReplay(s) {
			hint = "Click option or ←→ · Enter · V board · X replay · L log · H · Esc"
		}
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
	case s.Phase != phasePlay && !g.hideResult && !g.replayActive:
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

func (a *guiApp) drawPlayerPanel(dst *ebiten.Image, s *session, m *Match, pi int, isTop bool, pl Placement, hasPreview bool, now time.Time) {
	g, p := a.g, a.g.pal
	turn := !g.replayActive && s.Phase == phasePlay && m.Turn == pi
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
		a.text(dst, thinkingText(now), x, ty, 17, fontRegular, a.c(p.dim), text.AlignStart)
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
	if ft, fc := g.fairnessText(s); ft != "" {
		a.text(dst, truncPx(ft, 15, fontBold, w), x, y, 15, fontBold, a.c(fc), text.AlignStart)
		y += 26
	}
	toggleR := winMeterToggleRect()
	logBottom := toggleR.Y - 10
	showProb := g.save.Settings.WinMeter && !g.replayActive
	if showProb {
		logBottom -= 48
	}
	a.text(dst, "Log", x, y, 18, fontBold, a.c(p.accent), text.AlignStart)
	y += 28
	for i := len(s.Log) - 1; i >= 0 && y < logBottom; i-- {
		c := a.c(p.dim)
		if i == len(s.Log)-1 {
			c = a.c(p.fg)
		}
		for j, line := range wrapPx(g.logText(s, s.Log[i]), 15, fontRegular, w-16) {
			if y >= logBottom {
				break
			}
			if j == 0 {
				a.circle(dst, x+4, y, 2.5, c)
			}
			a.text(dst, line, x+14, y, 15, fontRegular, c, text.AlignStart)
			y += 21
		}
	}
	if showProb {
		pct, summary := g.winProbSummary(s)
		by := toggleR.Y - 46
		a.text(dst, truncPx("Win odds · "+summary, 14, fontBold, w), x, by+8, 14, fontBold, a.c(p.fg), text.AlignStart)
		bar := rect{x, by + 24, w, 10}
		a.fill(dst, bar, a.c(p.emptyBg), 5)
		if pct > 0 {
			fw := bar.W * float64(pct) / 100
			if fw < 6 {
				fw = 6
			}
			a.fill(dst, rect{bar.X, bar.Y, fw, bar.H}, a.c(p.win), 5)
		}
	}
	plabel := "Prob: Off [P]"
	a.fill(dst, toggleR, a.buttonBg(), 8)
	tc := a.c(p.dim)
	if g.save.Settings.WinMeter {
		plabel = "Prob: On [P]"
		tc = a.c(p.accent)
		a.stroke(dst, toggleR, tc, 8, 1.5)
	}
	a.text(dst, plabel, toggleR.cx(), toggleR.cy(), 14, fontBold, tc, text.AlignCenter)

	sndR := soundToggleRect()
	slabel := "Sound: Off [S]"
	a.fill(dst, sndR, a.buttonBg(), 8)
	sc := a.c(p.dim)
	if g.save.Settings.Sound {
		slabel = "Sound: On [S]"
		sc = a.c(p.accent)
		a.stroke(dst, sndR, sc, 8, 1.5)
	}
	a.text(dst, slabel, sndR.cx(), sndR.cy(), 14, fontBold, sc, text.AlignCenter)
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
	y := c.Y + 74
	for _, line := range lines {
		a.center(dst, line, y, 22, fontBold, a.c(mc))
		y += 28
	}
	sc := s.Match.Scores()
	a.center(dst, fmt.Sprintf("%s %d  –  %d %s", s.Names[0], sc[0], sc[1], s.Names[1]), y+4, 18, fontRegular, a.c(p.fg))
	y += 30
	if s.BestOf != 1 || s.FreePlay {
		a.center(dst, fmt.Sprintf("Rounds won %d – %d · %s", s.Wins[0], s.Wins[1], s.roundsLabel()), y, 16, fontRegular, a.c(p.dim))
		y += 26
	}
	if status, stc := g.resultStatus(s); status != "" {
		a.center(dst, status, y, 17, fontRegular, a.c(stc))
		y += 26
	}
	if ft, fc := g.fairnessText(s); ft != "" {
		a.center(dst, ft, y, 16, fontBold, a.c(fc))
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
