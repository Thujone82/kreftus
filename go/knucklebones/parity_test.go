package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func parityFirstLegal(m *Match) Move {
	for c := 0; c < gridCols; c++ {
		if mv := (Move{Col: c}); m.Legal(mv) {
			return mv
		}
	}
	return Move{}
}

func assertRectInViewport(t *testing.T, name string, r rect) {
	t.Helper()
	if r.W <= 0 || r.H <= 0 {
		t.Fatalf("%s has non-positive size: %+v", name, r)
	}
	if r.X < 0 || r.Y < 0 || r.X+r.W > baseW || r.Y+r.H > baseH {
		t.Fatalf("%s overflows %vx%v: %+v", name, baseW, baseH, r)
	}
}

func assertRectInside(t *testing.T, childName string, child rect, parentName string, parent rect) {
	t.Helper()
	if child.X < parent.X || child.Y < parent.Y || child.X+child.W > parent.X+parent.W || child.Y+child.H > parent.Y+parent.H {
		t.Fatalf("%s (%+v) overflows %s (%+v)", childName, child, parentName, parent)
	}
}

func assertNoRectOverlap(t *testing.T, rects map[string]rect) {
	t.Helper()
	names := make([]string, 0, len(rects))
	for k := range rects {
		names = append(names, k)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			a, b := rects[names[i]], rects[names[j]]
			if a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y {
				t.Fatalf("rects %s (%+v) and %s (%+v) overlap", names[i], a, names[j], b)
			}
		}
	}
}

func TestPhase45LayoutsDoNotOverlap(t *testing.T) {
	jc := joinCard()
	assertRectInViewport(t, "joinCard", jc)
	jf0 := joinFieldRect(0)
	jf1 := joinFieldRect(1)
	jbBack, jbOK := joinButtons()
	assertRectInside(t, "joinFieldRect(0)", jf0, "joinCard", jc)
	assertRectInside(t, "joinFieldRect(1)", jf1, "joinCard", jc)
	assertRectInside(t, "joinBack", jbBack, "joinCard", jc)
	assertRectInside(t, "joinOK", jbOK, "joinCard", jc)
	assertNoRectOverlap(t, map[string]rect{
		"joinField0": jf0,
		"joinField1": jf1,
		"joinBack":   jbBack,
		"joinOK":     jbOK,
	})

	g := newTestGame(t, 96, 34)
	g.openSettings(setupHost)
	hostRects := map[string]rect{"back": backButton()}
	for i, row := range g.settingRows() {
		r := settingsRowRect(i)
		if row == rowStart {
			r = settingsStartRect(i)
		}
		assertRectInViewport(t, fmt.Sprintf("hostRow(%d)", i), r)
		hostRects[fmt.Sprintf("row_%d", i)] = r
	}
	assertNoRectOverlap(t, hostRects)

	g.openSettings(setupOptions)
	optRects := map[string]rect{"back": backButton()}
	for i, row := range g.settingRows() {
		r := settingsRowRect(i)
		if row == rowDone {
			r = settingsStartRect(i)
		}
		assertRectInViewport(t, fmt.Sprintf("optRow(%d)", i), r)
		optRects[fmt.Sprintf("row_%d", i)] = r
	}
	assertNoRectOverlap(t, optRects)

	wt := winMeterToggleRect()
	stRect := soundToggleRect()
	assertRectInViewport(t, "winMeterToggleRect", wt)
	assertRectInViewport(t, "soundToggleRect", stRect)
	if wt.X < guiSideX || stRect.X < guiSideX {
		t.Fatalf("side panel toggle rects intrude left of x=%v: prob=%+v sound=%+v", guiSideX, wt, stRect)
	}
	assertNoRectOverlap(t, map[string]rect{"winProb": wt, "sound": stRect})
	playBtn0 := playButtons(4)[0]
	if wt.Y+wt.H > playBtn0.Y || stRect.Y+stRect.H > playBtn0.Y {
		t.Fatalf("side toggle rects overlap playButtons row (%+v): prob=%+v sound=%+v", playBtn0, wt, stRect)
	}

	// Verify replay and round-over play action buttons stay within <= 6 slots cleanly.
	g.beginGame(&offlineCtl{s: newSession(modeLocal, [2]string{"One", "Two"}, "", 1, Rules{}, testRNG()), rng: testRNG()})
	defer g.leaveGame()
	s := g.ctl.state()
	for s.Phase == phasePlay {
		g.ctl.place(parityFirstLegal(&s.Match))
	}
	if !canReplay(s) {
		t.Fatalf("expected canReplay true after round")
	}
	if acts := g.playActions(s); len(acts) > 6 {
		t.Fatalf("round-over playActions too many: %d", len(acts))
	} else {
		btns := playButtons(len(acts))
		for i, b := range btns {
			assertRectInViewport(t, fmt.Sprintf("endBtn(%d)", i), b)
		}
	}
	g.enterReplay(s)
	if acts := g.playActions(s); len(acts) < 3 || len(acts) > 6 {
		t.Fatalf("expected 3..6 replay actions, got %d (%+v)", len(acts), acts)
	} else {
		for i, b := range playButtons(len(acts)) {
			assertRectInViewport(t, fmt.Sprintf("replayBtn(%d)", i), b)
		}
	}
}

func TestOptionsAndThemesPersistence(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "save.json")
	g := newTestGame(t, 96, 34)
	g.savePath = savePath
	g.save = newSaveData()

	// Cycle all 4 themes via menuOptions left/right.
	g.menuIndex = menuOptions
	gotThemes := []string{g.save.Theme}
	for i := 0; i < len(themeOrder); i++ {
		g.handleKey(key(tcell.KeyRight))
		gotThemes = append(gotThemes, g.save.Theme)
	}
	if len(gotThemes) != 5 || gotThemes[0] != themeDark || gotThemes[1] != themeLight ||
		gotThemes[2] != themeContrast || gotThemes[3] != themeParchment || gotThemes[4] != themeDark {
		t.Fatalf("unexpected theme cycle order: %v", gotThemes)
	}
	for _, th := range themeOrder {
		p := paletteFor(th)
		if p.bg == 0 || p.fg == 0 || p.accent == 0 {
			t.Fatalf("palette %s has zero color fields", th)
		}
	}

	// Select Contrast theme and configure Options screen toggles.
	g.stepTheme(2) // dark -> contrast
	if g.save.Theme != themeContrast {
		t.Fatalf("expected %q, got %q", themeContrast, g.save.Theme)
	}
	g.handleKey(key(tcell.KeyEnter))
	if g.view != viewSettings || g.setupMode != setupOptions {
		t.Fatalf("expected setupOptions view, got view=%v mode=%q", g.view, g.setupMode)
	}

	// Row 0 = Theme (contrast), Row 1 = Sound (On -> Off), Row 2 = Colorblind (Off -> On), Row 3 = WinMeter (Off -> On), Row 4 = Done.
	g.handleKey(key(tcell.KeyDown))
	g.handleKey(key(tcell.KeyRight)) // Sound -> false
	g.handleKey(key(tcell.KeyDown))
	g.handleKey(key(tcell.KeyRight)) // Colorblind -> true
	g.handleKey(key(tcell.KeyDown))
	g.handleKey(key(tcell.KeyRight)) // WinMeter -> true
	g.handleKey(key(tcell.KeyDown))
	g.handleKey(key(tcell.KeyEnter)) // Done -> back to menu
	if g.view != viewMenu {
		t.Fatalf("expected return to menu on Done, got view=%v", g.view)
	}

	reloaded := loadSave(savePath)
	if reloaded.Theme != themeContrast {
		t.Fatalf("reloaded theme = %q, want %q", reloaded.Theme, themeContrast)
	}
	if reloaded.Settings.Sound {
		t.Fatalf("expected Sound=false persisted across reload")
	}
	if !reloaded.Settings.Colorblind || !reloaded.Settings.WinMeter {
		t.Fatalf("expected Colorblind=true and WinMeter=true persisted, got %+v", reloaded.Settings)
	}

	// Open Host setup, cycle BindIP, type a secret RoomCode, and verify RoomCode NEVER writes to disk.
	g.menuIndex = menuHost
	g.handleKey(key(tcell.KeyEnter))
	rows := g.settingRows()
	bindIdx, codeIdx := -1, -1
	for i, r := range rows {
		if r == rowBindIP {
			bindIdx = i
		}
		if r == rowRoomCode {
			codeIdx = i
		}
	}
	if bindIdx < 0 || codeIdx < 0 {
		t.Fatalf("missing rowBindIP or rowRoomCode in setupHost: %v", rows)
	}
	g.setIndex = bindIdx
	g.handleKey(key(tcell.KeyRight))
	if g.save.Settings.BindIP == "" {
		t.Fatalf("expected BindIP to cycle away from empty default")
	}
	g.setIndex = codeIdx
	g.focusSettingCaret()
	for _, r := range "KeepSecret99" {
		g.handleKey(runeKey(r))
	}
	if g.hostRoomCode != "KeepSecret99" {
		t.Fatalf("expected ephemeral hostRoomCode set, got %q", g.hostRoomCode)
	}
	g.persist()
	raw, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	if strings.Contains(string(raw), "KeepSecret99") {
		t.Fatalf("room code secret leaked into save file!")
	}
	if !strings.Contains(string(raw), `"sound": false`) && !strings.Contains(string(raw), `"sound":false`) {
		t.Fatalf("expected explicit sound:false in JSON, got %s", raw)
	}
}

func TestJoinTwoFieldWASDAndRoomCodeUI(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "save.json")
	host, err := startHost("Host", 0, hostConfig{
		BestOf:   1,
		BindIP:   "127.0.0.1",
		RoomCode: "secret42",
	}, testRNG())
	if err != nil {
		t.Fatalf("startHost: %v", err)
	}
	defer host.leave()

	g := newTestGame(t, 96, 34)
	g.savePath = savePath
	g.save = newSaveData()
	g.save.Name = "Guest"

	g.menuIndex = menuJoin
	g.handleKey(key(tcell.KeyEnter))
	if g.view != viewJoin || g.joinField != 0 {
		t.Fatalf("expected viewJoin field 0, got view=%v field=%d", g.view, g.joinField)
	}

	// Verify WASD inserts literal characters rather than hijacking field focus.
	prevInput := g.input
	for _, r := range "wasd" {
		g.handleKey(runeKey(r))
	}
	if g.input != prevInput+"wasd" || g.joinField != 0 {
		t.Fatalf("WASD broke address field input: input=%q want=%q field=%d", g.input, prevInput+"wasd", g.joinField)
	}
	g.handleKey(key(tcell.KeyTab))
	if g.joinField != 1 {
		t.Fatalf("Tab did not switch to room code field")
	}
	for _, r := range "wasd" {
		g.handleKey(runeKey(r))
	}
	if g.joinCode != "wasd" || g.joinField != 1 {
		t.Fatalf("WASD broke room code field input: joinCode=%q field=%d", g.joinCode, g.joinField)
	}

	// Replace with host address & wrong code first, confirm errBadAuth UI text, then retry with right code.
	g.focusJoinField(0)
	g.input = fmt.Sprintf("127.0.0.1:%d", host.hostPort())
	g.inputPos = runeLen(g.input)
	g.focusJoinField(1)
	g.joinCode = "wrongcode"
	g.inputPos = runeLen(g.joinCode)
	g.handleKey(key(tcell.KeyEnter))

	pumpUntil := func(done func() bool) bool {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			for drained := true; drained; {
				drained = false
				select {
				case ev := <-host.netEvents():
					host.handle(ev, time.Now())
					drained = true
				default:
				}
				if ch := g.ctl.netEvents(); ch != nil {
					select {
					case ev := <-ch:
						g.ctl.handle(ev, time.Now())
						g.syncView()
						drained = true
					default:
					}
				}
			}
			if done() {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return false
	}

	okErr := pumpUntil(func() bool {
		l, ok := g.ctl.(lobby)
		return ok && l.lobbyError() == errBadAuth
	})
	if !okErr {
		t.Fatalf("expected errBadAuth for wrong room code in UI flow")
	}
	g.render()
	if txt := screenText(g); !strings.Contains(txt, "Incorrect room code.") {
		t.Fatalf("lobby error screen missing room code message:\n%s", txt)
	}
	g.leaveGame()

	// Now connect with the right room code through keyJoin.
	g.menuIndex = menuJoin
	g.handleKey(key(tcell.KeyEnter))
	g.input = fmt.Sprintf("127.0.0.1:%d", host.hostPort())
	g.inputPos = runeLen(g.input)
	g.handleKey(key(tcell.KeyDown))
	for _, r := range "secret42" {
		g.handleKey(runeKey(r))
	}
	g.render()
	g.handleKey(key(tcell.KeyEnter))

	okPlay := pumpUntil(func() bool {
		return g.view == viewPlay && g.ctl.state() != nil && g.ctl.state().Phase == phasePlay
	})
	if !okPlay {
		t.Fatalf("failed to enter online play with valid room code; view=%v", g.view)
	}
	defer g.leaveGame()
	g.render()
	txt := screenText(g)
	if !strings.Contains(txt, "Fair play commit locked") {
		t.Fatalf("expected online guest HUD to show fair play commit status:\n%s", txt)
	}
	if raw, err := os.ReadFile(savePath); err == nil && (strings.Contains(string(raw), "secret42") || strings.Contains(string(raw), "wrongcode")) {
		t.Fatalf("join room code leaked to disk save")
	}
}

func TestWinProbReplayColorblindAndSounds(t *testing.T) {
	g := newTestGame(t, 96, 34)
	g.save.Settings.Colorblind = false
	g.save.Settings.WinMeter = false
	g.save.Settings.Sound = true

	// Build a controlled local game to verify colorblind runes (=, #, x), winProb, sound triggers, and replay.
	ctl := &offlineCtl{s: newSession(modeLocal, [2]string{"P1", "P2"}, "", 1, Rules{}, testRNG()), rng: testRNG()}
	g.beginGame(ctl)
	g.afterChange()
	g.rollAt = time.Now().Add(-time.Second)
	defer g.leaveGame()
	_ = g.drainSounds() // clear click sounds

	s := g.ctl.state()
	// Give P1 two 4s in col 0 and P2 a 5 in col 0, while P1 holds a 5 ready to aim at col 0 -> '=', 'x'.
	s.Match.Grids[0] = Grid{{4, 4, 0}, {0, 0, 0}, {0, 0, 0}}
	s.Match.Grids[1] = Grid{{5, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	s.Match.Turn = 0
	s.Match.Dice = [2]int{5, 0}
	g.cursor = 0

	// Toggle Colorblind (C) and WinMeter (P) via hotkeys.
	g.handleKey(runeKey('c'))
	g.handleKey(runeKey('p'))
	if !g.save.Settings.Colorblind || !g.save.Settings.WinMeter {
		t.Fatalf("expected C and P toggles enabled, got cb=%v wm=%v", g.save.Settings.Colorblind, g.save.Settings.WinMeter)
	}
	g.render()
	txt := screenText(g)
	if !strings.ContainsRune(txt, '=') {
		t.Fatalf("expected '=' colorblind pair mark in TUI grid:\n%s", txt)
	}
	if !strings.ContainsRune(txt, 'x') {
		t.Fatalf("expected 'x' colorblind doom mark on opposing die:\n%s", txt)
	}
	if !strings.Contains(txt, "Win odds") {
		t.Fatalf("expected Win probability meter in TUI side panel:\n%s", txt)
	}
	pct, summary := g.winProbSummary(s)
	if pct < 0 || pct > 100 || !strings.Contains(summary, "%") {
		t.Fatalf("unexpected winProbSummary: pct=%d summary=%q", pct, summary)
	}

	// Verify triple mark '#' as well.
	s.Match.Grids[0][1] = [3]int{6, 6, 6}
	g.render()
	if txt = screenText(g); !strings.ContainsRune(txt, '#') {
		t.Fatalf("expected '#' colorblind triple mark in TUI grid:\n%s", txt)
	}

	// Play move via Enter that destroys P2's 5 -> should emit sndDestroy.
	s.Match.Grids[0][1] = [3]int{}
	_ = g.drainSounds()
	g.handleKey(key(tcell.KeyEnter))
	g.afterChange()
	sounds := g.drainSounds()
	hasDestroy := false
	for _, id := range sounds {
		if id == sndDestroy {
			hasDestroy = true
		}
	}
	if !hasDestroy {
		t.Fatalf("expected sndDestroy when destroying opponent die, got %v", sounds)
	}
	// Press 's' during play to mute, verify overlay scrolling still preserves Sound, then press 's' to unmute.
	g.handleKey(runeKey('s'))
	if g.save.Settings.Sound {
		t.Fatalf("expected 's' in play to mute sound")
	}
	if reloaded := loadSave(g.savePath); reloaded.Settings.Sound {
		t.Fatalf("expected 's' mute persisted to disk")
	}
	g.render()
	if txt = screenText(g); !strings.Contains(txt, "Sound: Off [S]") {
		t.Fatalf("expected Sound: Off [S] in TUI side panel:\n%s", txt)
	}
	g.emitSound(sndPlace)
	if got := g.drainSounds(); len(got) != 0 {
		t.Fatalf("expected Sound=false to suppress emitSound, got %v", got)
	}
	// Inside Log overlay, 's' continues scrolling instead of toggling sound.
	g.handleKey(runeKey('l'))
	g.handleKey(runeKey('s'))
	if g.save.Settings.Sound {
		t.Fatalf("'s' inside Log overlay should scroll, not unmute sound")
	}
	g.handleKey(runeKey('l'))
	g.handleKey(runeKey('s'))
	if !g.save.Settings.Sound {
		t.Fatalf("expected second 's' in play to re-enable sound")
	}
	if got := g.drainSounds(); len(got) != 1 || got[0] != sndClick {
		t.Fatalf("expected sndClick confirmation when re-enabling sound via 's', got %v", got)
	}
	g.render()
	if txt = screenText(g); !strings.Contains(txt, "Sound: On [S]") {
		t.Fatalf("expected Sound: On [S] in TUI side panel:\n%s", txt)
	}

	// Complete the local round and verify Replay capability.
	_ = g.drainSounds()
	for s.Phase == phasePlay {
		g.ctl.place(parityFirstLegal(&s.Match))
	}
	g.afterChange()
	if s.Phase == phasePlay || !canReplay(s) {
		t.Fatalf("expected completed round with replay frames, phase=%v moves=%d", s.Phase, len(s.RoundMoves))
	}

	// Verify sndWin and sndLose trigger in single-player (modeAI) rounds.
	aiWin := &offlineCtl{s: newSession(modeAI, [2]string{"You", aiName[diffEasy]}, diffEasy, 1, Rules{}, testRNG()), rng: testRNG(), ai: true}
	g.beginGame(aiWin)
	g.afterChange()
	_ = g.drainSounds()
	aiWin.s.Match.Grids[0] = Grid{{6, 6, 6}, {6, 6, 6}, {6, 6, 0}}
	aiWin.s.Match.Grids[1] = Grid{{1, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	aiWin.s.Match.Turn = 0
	aiWin.s.Match.Dice = [2]int{6, 0}
	aiWin.place(Move{Col: 2})
	g.afterChange()
	if got := g.drainSounds(); len(got) == 0 || got[len(got)-1] != sndWin {
		t.Fatalf("expected sndWin on AI victory, got %v", got)
	}
	aiLoss := &offlineCtl{s: newSession(modeAI, [2]string{"You", aiName[diffEasy]}, diffEasy, 1, Rules{}, testRNG()), rng: testRNG(), ai: true}
	g.beginGame(aiLoss)
	g.afterChange()
	_ = g.drainSounds()
	aiLoss.s.Match.Grids[0] = Grid{{1, 1, 1}, {1, 1, 1}, {1, 1, 0}}
	aiLoss.s.Match.Grids[1] = Grid{{6, 6, 6}, {6, 6, 6}, {6, 6, 6}}
	aiLoss.s.Match.Turn = 0
	aiLoss.s.Match.Dice = [2]int{2, 0}
	aiLoss.place(Move{Col: 2})
	g.afterChange()
	if got := g.drainSounds(); len(got) == 0 || got[len(got)-1] != sndLose {
		t.Fatalf("expected sndLose on AI defeat, got %v", got)
	}

	// Switch back to finished local game to verify Replay scrubbing.
	g.beginGame(ctl)
	g.afterChange()
	// Hide result card and enter Replay with 'x'.
	g.handleKey(runeKey('x'))
	if !g.replayActive || g.replayStep != len(g.replayFrames)-1 {
		t.Fatalf("expected replay active at last step, active=%v step=%d", g.replayActive, g.replayStep)
	}
	totalSteps := len(g.replayFrames) - 1
	g.handleKey(key(tcell.KeyLeft))
	if g.replayStep != totalSteps-1 {
		t.Fatalf("expected step backward to %d, got %d", totalSteps-1, g.replayStep)
	}
	g.render()
	if txt = screenText(g); !strings.Contains(txt, fmt.Sprintf("REPLAY %d/%d", totalSteps-1, totalSteps)) {
		t.Fatalf("missing REPLAY banner on screen:\n%s", txt)
	}
	g.handleKey(key(tcell.KeyHome))
	if g.replayStep != 0 {
		t.Fatalf("Home should jump to step 0, got %d", g.replayStep)
	}
	g.handleKey(key(tcell.KeyEnd))
	if g.replayStep != totalSteps {
		t.Fatalf("End should jump to final step %d, got %d", totalSteps, g.replayStep)
	}
	g.handleKey(key(tcell.KeyEscape))
	if g.replayActive {
		t.Fatalf("Esc should exit replay mode")
	}
}

func TestCompactTUI52x20RendersCleanly(t *testing.T) {
	for _, dims := range [][2]int{{52, 20}, {60, 22}} {
		g := newTestGame(t, dims[0], dims[1])
		g.save.Settings.Colorblind = true
		g.save.Settings.WinMeter = true
		g.openSettings(setupHost)
		g.render()
		if txt := screenText(g); !strings.Contains(txt, "Room code") {
			t.Fatalf("%dx%d compact settings missing Room code row:\n%s", dims[0], dims[1], txt)
		}
		g.beginGame(&offlineCtl{s: newSession(modeLocal, [2]string{"Alpha", "Beta"}, "", 1, Rules{}, testRNG()), rng: testRNG()})
		g.render()
		txt := screenText(g)
		if !strings.Contains(txt, "KNUCKLEBONES") || !strings.Contains(txt, "Alpha") {
			t.Fatalf("%dx%d compact board missing essential labels:\n%s", dims[0], dims[1], txt)
		}
		g.leaveGame()
	}
}

func TestPCMAudioHeadlessDetection(t *testing.T) {
	cases := []struct {
		name        string
		goos        string
		ssh         bool
		sshCon      bool
		display     string
		wayland     string
		headlessEnv string
		want        bool
	}{
		{name: "windows_local", goos: "windows", want: true},
		{name: "windows_ssh", goos: "windows", ssh: true, want: false},
		{name: "windows_conpty_ssh", goos: "windows", sshCon: true, want: false},
		{name: "darwin_local", goos: "darwin", want: true},
		{name: "darwin_ssh", goos: "darwin", ssh: true, want: false},
		{name: "linux_x11", goos: "linux", display: ":0", want: true},
		{name: "linux_wayland", goos: "linux", wayland: "wayland-0", want: true},
		{name: "linux_ssh_with_display", goos: "linux", ssh: true, display: ":10.0", want: false},
		{name: "linux_headless_tty", goos: "linux", want: false},
		{name: "override_1", goos: "windows", headlessEnv: "1", want: false},
		{name: "override_true", goos: "darwin", headlessEnv: " True ", want: false},
		{name: "override_yes", goos: "linux", display: ":0", headlessEnv: "yes", want: false},
	}
	for _, tc := range cases {
		if got := needPCMAudio(tc.goos, tc.ssh, tc.sshCon, tc.display, tc.wayland, tc.headlessEnv); got != tc.want {
			t.Errorf("%s: needPCMAudio(...) = %v, want %v", tc.name, got, tc.want)
		}
	}

	t.Setenv("KNUCKLEBONES_HEADLESS", "1")
	if !headlessAudio() {
		t.Fatalf("expected KNUCKLEBONES_HEADLESS=1 to trigger headlessAudio()")
	}
	if !playPCMSound(sndNone) {
		t.Fatalf("expected playPCMSound(sndNone) to return true")
	}
	if playPCMSound(sndClick) {
		t.Fatalf("expected playPCMSound(sndClick) to return false when headless")
	}
}
