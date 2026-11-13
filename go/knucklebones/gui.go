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
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	xdraw "golang.org/x/image/draw"
)

const guiAvailable = true

//go:embed die.png
var diePNG []byte

type frameSig struct {
	w, h          int
	view          int
	menuIndex     int
	setIndex      int
	setupMode     string
	overlay       int
	overlayScroll int
	guideScroll   int
	resultIndex   int
	confirmIndex  int
	cursor        int
	dieSel        int
	hideResult    bool
	theme         string
	input         string
	inputPos      int
	note          string
	difficulty    string
	bestOf        int
	rules         Rules
	turnSeconds   int
	port          int
	bindIP        string
	hostRoomCode  string
	joinCode      string
	joinField     int
	netOnline     bool
	netFocus      int
	netRoomIdx    int
	netRoomScroll int
	netRoomsLen   int
	netUpdatedNs  int64
	netStatsKey   uint64
	sound         bool
	colorblind    bool
	winMeter      bool
	replayActive  bool
	replayStep    int
	fairness      int
	player2       string
	hoverTag      int
	clockTick     int64
	sessPhase     string
	sessRound     int
	sessSeries    int
	sessMoves     int
	sessTurn      int
	sessDice      [2]int
	sessAiming    bool
	sessAimCol    int
	sessAimDie    int
	sessConnected [2]bool
	sessVotes     [2]string
	sessForfeit   bool
	sessLogLen    int
	sessHistLen   int
	oppGone       bool
	lobbyErr       string
	lobbyPort      int
	lobbyBackFocus bool
	turnSecLeft   int
	discSecLeft   int
}

type guiApp struct {
	g             *game
	w, h          int
	vp            viewport
	logo          *ebiten.Image
	mouseX        float64
	mouseY        float64
	chars         []rune
	faceCache     map[faceKey]*text.GoTextFace
	dieCache      map[dieKey]*ebiten.Image
	dieScaleMilli int
	frame         *ebiten.Image
	lastSig       frameSig
	dirty         bool
}

func runGUI() error {
	src, err := png.Decode(bytes.NewReader(diePNG))
	if err != nil {
		return err
	}
	var icons []image.Image
	for _, n := range []int{16, 32, 48, 64, 128} {
		icons = append(icons, scaleImage(src, n))
	}
	logoImg := src
	if src.Bounds().Dx() != 256 || src.Bounds().Dy() != 256 {
		logoImg = scaleImage(src, 256)
	}
	icons = append(icons, logoImg)
	ebiten.SetWindowIcon(icons)
	ebiten.SetWindowTitle("Knucklebones")
	ebiten.SetWindowSize(1100, 760)
	ebiten.SetWindowSizeLimits(640, 440, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(false)

	a := &guiApp{g: newGame(), logo: ebiten.NewImageFromImage(logoImg), dirty: true}
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
					if g.ctl.handle(ev, now) {
						a.dirty = true
					}
				default:
					break drain
				}
			}
		}
		if g.ctl != nil && g.ctl.tick(now) {
			a.dirty = true
		}
	}
	if g.watcher != nil {
		select {
		case <-g.watcher.Notify():
			a.dirty = true
		default:
		}
	}
	g.syncView()
	a.pageSizes()
	for _, ev := range a.keyEvents() {
		a.dirty = true
		if g.handleKey(ev) {
			return ebiten.Termination
		}
		g.afterChange()
	}
	if a.mouse() {
		for _, id := range g.drainSounds() {
			_ = playPCMSound(id)
		}
		return ebiten.Termination
	}
	g.afterChange()
	for _, id := range g.drainSounds() {
		_ = playPCMSound(id)
	}
	return nil
}

func (a *guiApp) hoverTag(x, y float64) int {
	g := a.g
	switch g.view {
	case viewMenu:
		for i := 0; i < menuCount; i++ {
			if menuItemRect(i).has(x, y) {
				return 1 + i
			}
		}
	case viewSettings:
		if backButton().has(x, y) {
			return 1
		}
		for i, row := range g.settingRows() {
			if row == rowStart || row == rowDone {
				if settingsStartRect(i).has(x, y) {
					return 100 + i
				}
				continue
			}
			r := settingsRowRect(i)
			if r.has(x, y) {
				left, right := settingsArrows(r)
				if left.has(x, y) {
					return 200 + i*4 + 1
				}
				if right.has(x, y) {
					return 200 + i*4 + 2
				}
				return 200 + i*4
			}
		}
	case viewName:
		back, ok := promptButtons()
		if back.has(x, y) {
			return 1
		}
		if ok.has(x, y) {
			return 2
		}
	case viewNetwork:
		onBtn, dirBtn := netModeButtons()
		if onBtn.has(x, y) {
			return 10
		}
		if dirBtn.has(x, y) {
			return 11
		}
		if g.netOnline {
			back, join, host := netOnlineButtons()
			if back.has(x, y) {
				return 1
			}
			if join.has(x, y) {
				return 2
			}
			if host.has(x, y) {
				return 3
			}
			if netOnlineCodeRect().has(x, y) {
				return 4
			}
			snap := g.networkSnapshot()
			for i := 0; i < netVisibleRooms && g.netRoomScroll+i < len(snap.Rooms); i++ {
				if netRoomRowRect(i).has(x, y) {
					return 20 + i
				}
			}
		} else {
			if netDirectHostButton().has(x, y) {
				return 3
			}
			back, ok := netDirectButtons()
			if back.has(x, y) {
				return 1
			}
			if ok.has(x, y) {
				return 2
			}
			if netDirectFieldRect(0).has(x, y) {
				return 5
			}
			if netDirectFieldRect(1).has(x, y) {
				return 4
			}
		}
	case viewLobby:
		l, _ := g.ctl.(lobby)
		retry := l != nil && lobbyCanRetry(l.lobbyError())
		back, again := lobbyButtons(retry)
		if back.has(x, y) {
			return 1
		}
		if retry && again.has(x, y) {
			return 2
		}
	case viewGuide, viewStats:
		if backButton().has(x, y) {
			return 1
		}
	case viewPlay:
		if g.ctl == nil || g.ctl.state() == nil {
			return 0
		}
		s := g.ctl.state()
		switch g.overlay {
		case overlayResign, overlayLeave:
			for i, b := range confirmButtons(confirmCard(a.confirmLines(s))) {
				if b.has(x, y) {
					return 300 + i
				}
			}
			return 0
		case overlayLog, overlayHistory:
			if overlayClose().has(x, y) {
				return 350
			}
			return 0
		}
		for i, b := range playButtons(len(g.playActions(s))) {
			if b.has(x, y) {
				return 400 + i
			}
		}
		if winMeterToggleRect().has(x, y) {
			return 480
		}
		if soundToggleRect().has(x, y) {
			return 481
		}
		if g.replayActive {
			return 0
		}
		if s.Phase != phasePlay && !g.hideResult {
			card := resultCard(a.resultLines(s))
			for i, b := range resultButtons(card, len(g.resultChoices(s))) {
				if b.has(x, y) {
					return 500 + i
				}
			}
		}
		if g.myTurn(s) && !g.rolling() {
			if c := columnAt(x, y); c >= 0 {
				return 600 + c
			}
			if s.Match.Dice[1] != 0 {
				isTop := g.moverOnTop(s)
				for i := 0; i < 2; i++ {
					if trayDieRect(isTop, i).has(x, y) {
						return 700 + i
					}
				}
			}
		}
	}
	return 0
}

func (a *guiApp) frameSignature(now time.Time) (frameSig, bool) {
	g := a.g
	st := g.save.Settings
	sig := frameSig{
		w:             a.w,
		h:             a.h,
		view:          g.view,
		menuIndex:     g.menuIndex,
		setIndex:      g.setIndex,
		setupMode:     g.setupMode,
		overlay:       g.overlay,
		overlayScroll: g.overlayScroll,
		guideScroll:   g.guideScroll,
		resultIndex:   g.resultIndex,
		confirmIndex:  g.confirmIndex,
		cursor:        g.cursor,
		dieSel:        g.dieSel,
		hideResult:    g.hideResult,
		theme:         g.save.Theme,
		input:         g.input,
		inputPos:      g.inputPos,
		note:          g.note,
		difficulty:    st.Difficulty,
		bestOf:        st.BestOf,
		rules:         st.Rules,
		turnSeconds:   st.TurnSeconds,
		port:          st.Port,
		bindIP:        st.BindIP,
		hostRoomCode:  g.hostRoomCode,
		joinCode:      g.joinCode,
		joinField:     g.joinField,
		netOnline:     g.netOnline,
		netFocus:      g.netFocus,
		netRoomIdx:    g.netRoomIdx,
		netRoomScroll: g.netRoomScroll,
		sound:         st.Sound,
		colorblind:    st.Colorblind,
		winMeter:      st.WinMeter,
		replayActive:  g.replayActive,
		replayStep:    g.replayStep,
		player2:       st.Player2,
		hoverTag:      a.hoverTag(a.mouseX, a.mouseY),
	}
	anim := false
	switch g.view {
	case viewName:
		sig.clockTick = now.UnixMilli() / 500
	case viewNetwork:
		if g.netFocus == netFocusAddr || g.netFocus == netFocusCode {
			sig.clockTick = now.UnixMilli() / 500
		}
		if g.netOnline {
			snap := g.networkSnapshot()
			sig.netRoomsLen = len(snap.Rooms)
			sig.netUpdatedNs = snap.UpdatedAt.UnixNano()
			sig.netStatsKey = uint64(snap.Stats.LiveMatches)*1000003 + snap.Stats.CompletedMatches*1009 + snap.Stats.TotalDicePlayed
			if snap.Err != "" {
				sig.lobbyErr = snap.Err
			}
			if len(snap.Rooms) > 0 && g.netFocus != netFocusCode {
				sig.clockTick = now.Unix()
			}
		}
	case viewSettings:
		rows := g.settingRows()
		if g.setIndex >= 0 && g.setIndex < len(rows) && (rows[g.setIndex] == rowPlayer2 || rows[g.setIndex] == rowPort || rows[g.setIndex] == rowRoomCode) {
			sig.clockTick = now.UnixMilli() / 500
		}
	case viewLobby:
		if l, ok := g.ctl.(lobby); ok {
			sig.lobbyErr = l.lobbyError()
			sig.lobbyPort = l.hostPort()
			sig.lobbyBackFocus = g.lobbyBackFocus
			if sig.lobbyPort > 0 || l.hostRoomID() != "" {
				sig.clockTick = int64(time.Since(l.waitingSince()).Seconds())
			} else if sig.lobbyErr == "" {
				sig.clockTick = now.UnixMilli() / 400
			}
		}
	case viewPlay:
		if g.ctl != nil {
			sig.oppGone = g.ctl.opponentGone()
			sig.fairness = g.ctl.fairness()
			if s := g.ctl.state(); s != nil {
				sig.sessPhase = s.Phase
				sig.sessRound = s.Round
				sig.sessSeries = s.Series
				sig.sessMoves = s.Match.Moves
				sig.sessTurn = s.Match.Turn
				sig.sessDice = s.Match.Dice
				sig.sessAiming = s.Aiming
				sig.sessAimCol = s.AimCol
				sig.sessAimDie = s.AimDie
				sig.sessConnected = s.Connected
				sig.sessVotes = s.Votes
				sig.sessForfeit = s.Forfeit != nil
				sig.sessLogLen = len(s.Log)
				sig.sessHistLen = len(s.History)
				if left := g.ctl.turnLeft(now); left > 0 {
					sig.turnSecLeft = int(left.Seconds() + 0.999)
				}
				if p, d := g.ctl.disconnectLeft(now); p >= 0 {
					sig.discSecLeft = p*10000 + int(d.Seconds()+0.999)
				}
				if s.Phase == phasePlay && !g.myTurn(s) && !g.rolling() && !g.replayActive {
					sig.clockTick = now.UnixMilli() / 350
				}
			}
			if g.rolling() || effectProgress(g.effectAt, now) >= 0 {
				anim = true
			}
		}
	}
	return sig, anim
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
	{ebiten.KeyEnd, tcell.KeyEnd},
}

// repeating reports a fresh press or an auto-repeat of a held key.
func repeating(k ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(k) {
		return true
	}
	d := inpututil.KeyPressDuration(k)
	return d >= 24 && (d-24)%4 == 0
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
			if row == rowStart || row == rowDone {
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
			if row == rowPlayer2 || row == rowPort || row == rowRoomCode {
				g.focusSettingCaret()
				field := settingsTextField(r)
				if field.has(x, y) {
					value := g.save.Settings.Player2
					switch row {
					case rowPort:
						value = g.portEditString()
					case rowRoomCode:
						value = g.hostRoomCode
					}
					g.inputPos = caretIndexAt(value, x, field.X+12, 17, fontRegular)
				}
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
	case viewName:
		if !click {
			return false
		}
		f := promptField()
		if f.has(x, y) {
			g.inputPos = caretIndexAt(g.input, x, f.X+12, 18, fontRegular)
			return false
		}
		back, ok := promptButtons()
		switch {
		case back.has(x, y):
			return send(keyEv(tcell.KeyEscape))
		case ok.has(x, y):
			return send(keyEv(tcell.KeyEnter))
		}
	case viewNetwork:
		if g.netOnline && wheel != 0 && netRoomsRect().has(x, y) {
			snap := g.networkSnapshot()
			maxScroll := len(snap.Rooms) - netVisibleRooms
			if maxScroll < 0 {
				maxScroll = 0
			}
			g.netRoomScroll -= int(wheel)
			if g.netRoomScroll < 0 {
				g.netRoomScroll = 0
			}
			if g.netRoomScroll > maxScroll {
				g.netRoomScroll = maxScroll
			}
		}
		if !click {
			return false
		}
		onBtn, dirBtn := netModeButtons()
		if onBtn.has(x, y) {
			if !g.netOnline {
				g.openNetwork(true)
			}
			return false
		}
		if dirBtn.has(x, y) {
			if g.netOnline {
				g.openNetwork(false)
			}
			return false
		}
		if g.netOnline {
			snap := g.networkSnapshot()
			for i := 0; i < netVisibleRooms && g.netRoomScroll+i < len(snap.Rooms); i++ {
				if netRoomRowRect(i).has(x, y) {
					idx := g.netRoomScroll + i
					if g.netRoomIdx == idx && g.netFocus == netFocusRooms {
						g.joinSelectedOnlineRoom()
					} else {
						g.netRoomIdx = idx
						g.focusNetworkField(netFocusRooms)
					}
					return false
				}
			}
			cf := netOnlineCodeRect()
			if cf.has(x, y) {
				g.focusNetworkField(netFocusCode)
				g.inputPos = caretIndexAt(g.joinCode, x, cf.X+12, 18, fontRegular)
				return false
			}
			back, join, host := netOnlineButtons()
			switch {
			case back.has(x, y):
				return send(keyEv(tcell.KeyEscape))
			case join.has(x, y):
				g.joinSelectedOnlineRoom()
				return false
			case host.has(x, y):
				g.stopWatcher()
				g.openSettings(setupOnlineHost)
				return false
			}
			return false
		}
		if netDirectHostButton().has(x, y) {
			g.openSettings(setupHost)
			return false
		}
		for i := 0; i < 2; i++ {
			f := netDirectFieldRect(i)
			if f.has(x, y) {
				if i == 0 {
					g.focusNetworkField(netFocusAddr)
					g.inputPos = caretIndexAt(g.input, x, f.X+12, 18, fontRegular)
				} else {
					g.focusNetworkField(netFocusCode)
					g.inputPos = caretIndexAt(g.joinCode, x, f.X+12, 18, fontRegular)
				}
				return false
			}
		}
		back, ok := netDirectButtons()
		switch {
		case back.has(x, y):
			return send(keyEv(tcell.KeyEscape))
		case ok.has(x, y):
			g.joinDirect()
			return false
		}
	case viewLobby:
		if !click {
			return false
		}
		l, _ := g.ctl.(lobby)
		retry := l != nil && lobbyCanRetry(l.lobbyError())
		back, again := lobbyButtons(retry)
		switch {
		case back.has(x, y):
			return send(keyEv(tcell.KeyEscape))
		case retry && again.has(x, y):
			g.lobbyBackFocus = false
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
		if winMeterToggleRect().has(x, y) {
			return send(runeEv('p'))
		}
		if soundToggleRect().has(x, y) {
			return send(runeEv('s'))
		}
	}

	if g.replayActive {
		return false
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
