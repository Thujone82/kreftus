//go:build windows || darwin || (linux && cgo)

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	xdraw "golang.org/x/image/draw"
)

const guiAvailable = true

//go:embed die.png
var diePNG []byte

type guiApp struct {
	g      *game
	w, h   int
	vp     viewport
	logo   *ebiten.Image
	mouseX float64
	mouseY float64
	chars  []rune
}

func runGUI() error {
	src, err := png.Decode(bytes.NewReader(diePNG))
	if err != nil {
		return err
	}
	var icons []image.Image
	for _, n := range []int{16, 32, 48, 64, 128, 256} {
		icons = append(icons, scaleImage(src, n))
	}
	ebiten.SetWindowIcon(icons)
	ebiten.SetWindowTitle("Knucklebones")
	ebiten.SetWindowSize(1100, 760)
	ebiten.SetWindowSizeLimits(640, 440, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(true)

	a := &guiApp{g: newGame(), logo: ebiten.NewImageFromImage(scaleImage(src, 256))}
	a.g.width, a.g.height = 80, 30
	defer a.g.leaveGame()
	initFonts()
	err = ebiten.RunGame(a)
	if err == ebiten.Termination {
		return nil
	}
	return err
}

func scaleImage(src image.Image, n int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func (a *guiApp) Layout(outsideW, outsideH int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	a.w, a.h = int(float64(outsideW)*s), int(float64(outsideH)*s)
	a.vp = fitViewport(float64(a.w), float64(a.h))
	return a.w, a.h
}

func (a *guiApp) Update() error {
	g := a.g
	if ebiten.IsWindowBeingClosed() {
		return ebiten.Termination
	}
	now := time.Now()
	if g.ctl != nil {
		if ch := g.ctl.netEvents(); ch != nil {
		drain:
			for {
				select {
				case ev := <-ch:
					if g.ctl == nil {
						break drain
					}
					g.ctl.handle(ev, now)
				default:
					break drain
				}
			}
		}
		if g.ctl != nil {
			g.ctl.tick(now)
		}
	}
	g.syncView()
	a.pageSizes()
	for _, ev := range a.keyEvents() {
		if g.handleKey(ev) {
			return ebiten.Termination
		}
		g.afterChange()
	}
	if a.mouse() {
		return ebiten.Termination
	}
	g.afterChange()
	return nil
}

// pageSizes sets width and height so PgUp/PgDn page by what the window shows.
func (a *guiApp) pageSizes() {
	switch a.g.view {
	case viewGuide:
		a.g.height = guiGuideRoom + 4
	default:
		a.g.height = guiOverRoom + 6
	}
}

// ---- keyboard ----

var keyMap = []struct {
	k ebiten.Key
	t tcell.Key
}{
	{ebiten.KeyArrowUp, tcell.KeyUp},
	{ebiten.KeyArrowDown, tcell.KeyDown},
	{ebiten.KeyArrowLeft, tcell.KeyLeft},
	{ebiten.KeyArrowRight, tcell.KeyRight},
	{ebiten.KeyEnter, tcell.KeyEnter},
	{ebiten.KeyNumpadEnter, tcell.KeyEnter},
	{ebiten.KeyEscape, tcell.KeyEscape},
	{ebiten.KeyTab, tcell.KeyTab},
	{ebiten.KeyBackspace, tcell.KeyBackspace2},
	{ebiten.KeyDelete, tcell.KeyDelete},
	{ebiten.KeyPageUp, tcell.KeyPgUp},
	{ebiten.KeyPageDown, tcell.KeyPgDn},
	{ebiten.KeyHome, tcell.KeyHome},
}

// repeating reports a fresh press or an auto-repeat of a held key.
func repeating(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d >= 24 && (d-24)%4 == 0)
}

func (a *guiApp) keyEvents() []*tcell.EventKey {
	var out []*tcell.EventKey
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	if ctrl {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyC):
			out = append(out, keyEv(tcell.KeyCtrlC))
		case inpututil.IsKeyJustPressed(ebiten.KeyU):
			out = append(out, keyEv(tcell.KeyCtrlU))
		}
	}
	for _, m := range keyMap {
		if !repeating(m.k) {
			continue
		}
		t := m.t
		if t == tcell.KeyTab && shift {
			t = tcell.KeyBacktab
		}
		out = append(out, keyEv(t))
	}
	if !ctrl {
		a.chars = ebiten.AppendInputChars(a.chars[:0])
		for _, r := range a.chars {
			out = append(out, runeEv(r))
		}
	}
	return out
}

// ---- mouse ----

// mouse handles hover and clicks; it returns true to quit.
func (a *guiApp) mouse() bool {
	g := a.g
	cx, cy := ebiten.CursorPosition()
	x, y := a.vp.toBase(float64(cx), float64(cy))
	moved := x != a.mouseX || y != a.mouseY
	a.mouseX, a.mouseY = x, y
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	_, wheel := ebiten.Wheel()

	send := func(ev *tcell.EventKey) bool {
		if ev == nil {
			return false
		}
		quit := g.handleKey(ev)
		g.afterChange()
		return quit
	}

	switch g.view {
	case viewMenu:
		for i := 0; i < menuCount; i++ {
			if menuItemRect(i).has(x, y) {
				if moved {
					g.menuIndex = i
				}
				if click {
					g.menuIndex = i
					return send(keyEv(tcell.KeyEnter))
				}
			}
		}
	case viewSettings:
		if !click {
			return false
		}
		if backButton().has(x, y) {
			return send(keyEv(tcell.KeyEscape))
		}
		for i, row := range g.settingRows() {
			if row == rowStart {
				if settingsStartRect(i).has(x, y) {
					g.setIndex = i
					return send(keyEv(tcell.KeyEnter))
				}
				continue
			}
			r := settingsRowRect(i)
			if !r.has(x, y) {
				continue
			}
			g.setIndex = i
			if row == rowPlayer2 || row == rowPort {
				return false
			}
			left, right := settingsArrows(r)
			if left.has(x, y) {
				return send(keyEv(tcell.KeyLeft))
			}
			if right.has(x, y) {
				return send(keyEv(tcell.KeyRight))
			}
			return send(settingClick(row))
		}
	case viewName, viewJoin:
		if !click {
			return false
		}
		back, ok := promptButtons()
		switch {
		case back.has(x, y):
			return send(keyEv(tcell.KeyEscape))
		case ok.has(x, y):
			return send(keyEv(tcell.KeyEnter))
		}
	case viewLobby:
		if !click {
			return false
		}
		l, _ := g.ctl.(lobby)
		retry := l != nil && l.lobbyError() != ""
		back, again := lobbyButtons(retry)
		switch {
		case back.has(x, y):
			return send(keyEv(tcell.KeyEscape))
		case retry && again.has(x, y):
			return send(keyEv(tcell.KeyEnter))
		}
	case viewGuide:
		if wheel != 0 {
			g.guideScroll -= int(wheel * 3)
			if g.guideScroll < 0 {
				g.guideScroll = 0
			}
		}
		if click && backButton().has(x, y) {
			return send(keyEv(tcell.KeyEscape))
		}
	case viewStats:
		if click && backButton().has(x, y) {
			return send(keyEv(tcell.KeyEscape))
		}
	case viewPlay:
		return a.mousePlay(x, y, moved, click, wheel, send)
	}
	return false
}

func (a *guiApp) mousePlay(x, y float64, moved, click bool, wheel float64, send func(*tcell.EventKey) bool) bool {
	g := a.g
	s := g.ctl.state()
	if s == nil {
		return false
	}
	switch g.overlay {
	case overlayResign, overlayLeave:
		if !click {
			return false
		}
		for i, b := range confirmButtons(confirmCard(a.confirmLines(s))) {
			if b.has(x, y) {
				g.confirmIndex = i
				return send(keyEv(tcell.KeyEnter))
			}
		}
		return false
	case overlayLog, overlayHistory:
		if wheel != 0 {
			step := int(wheel * 3)
			if g.overlay == overlayHistory {
				step = -step
			}
			g.overlayScroll += step
			if g.overlayScroll < 0 {
				g.overlayScroll = 0
			}
		}
		if click && overlayClose().has(x, y) {
			return send(keyEv(tcell.KeyEscape))
		}
		return false
	}

	if click {
		for i, b := range playButtons(len(g.playActions(s))) {
			if b.has(x, y) {
				return send(g.playActions(s)[i].key)
			}
		}
	}

	if s.Phase != phasePlay {
		if !click || g.hideResult {
			return false
		}
		card := resultCard(a.resultLines(s))
		for i, b := range resultButtons(card, len(g.resultChoices(s))) {
			if b.has(x, y) {
				g.resultIndex = i
				return send(keyEv(tcell.KeyEnter))
			}
		}
		return false
	}

	if !g.myTurn(s) || g.rolling() {
		return false
	}
	if s.Match.Dice[1] != 0 && click {
		isTop := g.moverOnTop(s)
		for i := 0; i < 2; i++ {
			if trayDieRect(isTop, i).has(x, y) {
				g.dieSel = i
				return false
			}
		}
	}
	if c := columnAt(x, y); c >= 0 {
		if moved {
			g.cursor = c
		}
		if click {
			g.cursor = c
			g.placeAtCursor(s)
			g.afterChange()
		}
	}
	return false
}

// moverOnTop is whether the player to move sits on the top board.
func (g *game) moverOnTop(s *session) bool {
	bottom := g.ctl.seat()
	if bottom < 0 {
		bottom = 0
	}
	return s.Match.Turn != bottom
}
