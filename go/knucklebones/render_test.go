package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func newTestGame(t *testing.T, w, h int) *game {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	sim.SetSize(w, h)
	t.Cleanup(sim.Fini)
	sv := newSaveData()
	sv.Name = "Kaleb"
	g := &game{
		screen:   sim,
		width:    w,
		height:   h,
		save:     sv,
		savePath: filepath.Join(t.TempDir(), saveFileName),
		rng:      testRNG(),
	}
	g.pal = themes[themeDark]
	return g
}

func screenText(g *game) string {
	sim := g.screen.(tcell.SimulationScreen)
	cells, w, h := sim.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := cells[y*w+x].Runes
			if len(r) == 0 {
				b.WriteRune(' ')
			} else {
				b.WriteRune(r[0])
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, tcell.ModNone) }
func runeKey(r rune) *tcell.EventKey  { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

func dump(t *testing.T, g *game, name string) string {
	g.render()
	text := screenText(g)
	if os.Getenv("KB_DUMP") != "" {
		t.Logf("---- %s ----\n%s", name, text)
	}
	return text
}

func TestRenderEveryView(t *testing.T) {
	for _, size := range [][2]int{{80, 30}, {60, 22}, {120, 40}} {
		g := newTestGame(t, size[0], size[1])
		g.netAddr = "127.0.0.1:1"
		for _, v := range []int{viewMenu, viewName, viewNetwork, viewGuide, viewStats} {
			g.view = v
			g.netOnline = true
			dump(t, g, "view")
			if v == viewNetwork {
				g.netOnline = false
				dump(t, g, "view direct")
			}
		}
		for _, m := range []string{modeAI, modeLocal, setupHost, setupOnlineHost} {
			g.openSettings(m)
			dump(t, g, "settings "+m)
		}
	}
}

func TestPlayAgainstAIThroughUI(t *testing.T) {
	g := newTestGame(t, 80, 30)
	g.view = viewMenu
	g.menuIndex = menuAI
	g.handleKey(key(tcell.KeyEnter))
	if g.view != viewSettings {
		t.Fatalf("view %d, want settings", g.view)
	}
	dump(t, g, "settings")
	g.handleKey(key(tcell.KeyEnter)) // Start game is preselected
	if g.view != viewPlay || g.ctl == nil {
		t.Fatalf("view %d, want play", g.view)
	}
	g.afterChange()
	ctl := g.ctl.(*offlineCtl)
	ctl.s.Match.Turn = 0
	ctl.s.Match.Dice = [2]int{5, 0}
	ctl.s.Match.Grids[1][1] = [gridRows]int{5, 5}
	g.rollAt = time.Now().Add(-time.Second)
	g.cursor = 1
	text := dump(t, g, "preview")
	if !strings.Contains(text, "destroys 2") {
		t.Fatalf("preview missing destroy text:\n%s", text)
	}
	g.handleKey(runeKey('2'))
	if ctl.s.Match.Grids[1].Count(1) != 0 {
		t.Fatal("placing did not destroy the facing 5s")
	}
	for i := 0; i < 200 && ctl.s.Phase == phasePlay; i++ {
		if ctl.s.Match.Turn == 1 {
			ctl.aiAt = time.Time{}
			ctl.tick(time.Now())
			ctl.tick(time.Now().Add(2 * aiDelay))
		} else {
			g.rollAt = time.Now().Add(-time.Second)
			mv := ctl.s.Match.LegalMoves()[0]
			g.cursor = mv.Col
			g.handleKey(key(tcell.KeyEnter))
		}
		g.afterChange()
	}
	if ctl.s.Phase == phasePlay {
		t.Fatal("game did not finish")
	}
	text = dump(t, g, "result")
	if !strings.Contains(text, "Play another game") && !strings.Contains(text, "Continue game") {
		t.Fatalf("result buttons missing:\n%s", text)
	}
	r := g.save.Records["ai-medium"]
	if r == nil || r.Games != 1 || r.Wins+r.Losses+r.Draws != 1 {
		t.Fatalf("record %+v", r)
	}
	g.handleKey(runeKey('l'))
	dump(t, g, "log")
	g.handleKey(runeKey('h'))
	dump(t, g, "history")
	g.handleKey(key(tcell.KeyEscape))
	g.view = viewStats
	dump(t, g, "stats")
}

func TestLocalResignAndTwoDice(t *testing.T) {
	g := newTestGame(t, 80, 30)
	g.save.Settings.Rules.TwoDice = true
	g.save.Settings.BestOf = 3
	g.openSettings(modeLocal)
	g.startFromSettings()
	g.afterChange()
	g.rollAt = time.Now().Add(-time.Second)
	dump(t, g, "two dice")
	turn := g.ctl.state().Match.Turn
	g.handleKey(runeKey('r'))
	if g.overlay != overlayResign {
		t.Fatal("resign prompt did not open")
	}
	dump(t, g, "resign")
	g.handleKey(runeKey('y'))
	s := g.ctl.state()
	if s.Forfeit == nil || s.Forfeit.Loser != turn {
		t.Fatalf("forfeit %+v", s.Forfeit)
	}
	g.afterChange()
	dump(t, g, "forfeit")
}

func TestLightThemeAndLobby(t *testing.T) {
	g := newTestGame(t, 80, 30)
	g.toggleTheme()
	if g.save.Theme != themeLight {
		t.Fatal("theme did not toggle")
	}
	// Bind an ephemeral port so a leftover process on 8647 cannot flake this test.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	g.save.Settings.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	g.openSettings(setupHost)
	g.startFromSettings()
	if g.view != viewLobby {
		t.Fatalf("view %d, want lobby (note %q)", g.view, g.note)
	}
	dump(t, g, "host lobby")
	g.leaveGame()

	g.beginGame(startClient("Kaleb", "127.0.0.1:1"))
	deadline := time.Now().Add(5 * time.Second)
	for g.ctl.(lobby).lobbyError() == "" && time.Now().Before(deadline) {
		select {
		case ev := <-g.ctl.netEvents():
			g.ctl.handle(ev, time.Now())
		case <-time.After(20 * time.Millisecond):
		}
	}
	dump(t, g, "join error")
	g.leaveGame()
}

func TestNumpadPlacementAndChallengeCelebration(t *testing.T) {
	// 1. Verify playColDigit maps top-row 1..3 and numpad End/Down/PgDn keys.
	cases := []struct {
		ev   *tcell.EventKey
		want int
	}{
		{runeKey('1'), 0},
		{runeKey('2'), 1},
		{runeKey('3'), 2},
		{key(tcell.KeyEnd), 0},
		{key(tcell.KeyDownLeft), 0},
		{key(tcell.KeyDown), 1},
		{key(tcell.KeyPgDn), 2},
		{key(tcell.KeyDownRight), 2},
	}
	for _, tc := range cases {
		col, ok := playColDigit(tc.ev)
		if !ok || col != tc.want {
			t.Fatalf("playColDigit(%v) = (%d, %v), want (%d, true)", tc.ev.Name(), col, ok, tc.want)
		}
	}

	// Verify numpad keys actually place dice during local play.
	g := newTestGame(t, 80, 28)
	g.openSettings(modeLocal)
	g.startFromSettings()
	g.rollAt = time.Now().Add(-time.Second)
	for _, k := range []tcell.Key{tcell.KeyEnd, tcell.KeyDown, tcell.KeyPgDn} {
		g.rollAt = time.Now().Add(-time.Second)
		before := g.ctl.state().Match.Moves
		g.handleKey(key(k))
		if got := g.ctl.state().Match.Moves; got != before+1 {
			t.Fatalf("expected move count %d after key %v, got %d", before+1, k, got)
		}
	}

	// 2. Verify Challenge Top 10 celebration triggers, plays sndCelebrate, renders celebration overlay,
	// and transitions to the full 3-match breakdown table on dismissal.
	s := g.ctl.state()
	s.Challenge = true
	s.Names = [2]string{"Shrumy", "Thujone"}
	s.Phase = phaseGameOver
	s.Round = 3
	s.Series = 1
	s.ChallengeRank = 4
	s.ChallengeDelta = 68
	s.ChallengeElapsedMs = 108900
	s.Wins = [2]int{1, 2}
	s.History = []historyEntry{
		{Round: 1, Scores: [2]int{45, 70}, Winner: 1},
		{Round: 2, Scores: [2]int{62, 50}, Winner: 0},
		{Round: 3, Scores: [2]int{30, 85}, Winner: 1},
	}
	g.save.Settings.Sound = true
	g.soundQueue = nil
	g.afterChange()

	if !g.isChallengeCelebrating(s, time.Now()) {
		t.Fatal("expected Challenge Top 10 celebration to be active")
	}
	sounds := g.drainSounds()
	foundCeleb := false
	for _, id := range sounds {
		if id == sndCelebrate {
			foundCeleb = true
		}
	}
	if !foundCeleb {
		t.Fatalf("expected sndCelebrate in sound queue, got %v", sounds)
	}
	celebTxt := screenText(g)
	if !strings.Contains(celebTxt, "TOP 10 LEADERBOARD") || !strings.Contains(celebTxt, "Congratulations, Thujone!") {
		t.Fatalf("celebration screen missing expected text:\n%s", celebTxt)
	}

	// Past the 350ms debounce, pressing Enter dismisses the celebration and shows the breakdown table.
	g.challengeCelebAt = time.Now().Add(-500 * time.Millisecond)
	g.handleKey(key(tcell.KeyEnter))
	g.afterChange()
	if g.isChallengeCelebrating(s, time.Now()) {
		t.Fatal("expected celebration to be dismissed after Enter")
	}
	resTxt := screenText(g)
	for _, want := range []string{"Challenge Complete", "Top 10 Rank #4", "Match 1", "Match 2", "Match 3", "Combined"} {
		if !strings.Contains(resTxt, want) {
			t.Fatalf("challenge result breakdown missing %q:\n%s", want, resTxt)
		}
	}
}

func TestExeOptionsCompletenessChallengeAccessAndPreviewPlace(t *testing.T) {
	g := newTestGame(t, 80, 28)

	// 1. Complete Options rows in exe match PWA parity set.
	g.openOptionsFrom(viewMenu)
	wantRows := []int{rowTheme, rowSound, rowQuickPlace, rowColorblind, rowWinMeter, rowPlayerName, rowPlayer2, rowDone}
	gotRows := g.settingRows()
	if len(gotRows) != len(wantRows) {
		t.Fatalf("setupOptions rows len = %d, want %d (%v)", len(gotRows), len(wantRows), gotRows)
	}
	for i := range wantRows {
		if gotRows[i] != wantRows[i] {
			t.Fatalf("setupOptions row[%d] = %d, want %d", i, gotRows[i], wantRows[i])
		}
	}

	// 2. Challenge lobby offers Options [O] instead of Change Name [N] and returns to Challenge lobby.
	g.view = viewNetwork
	g.netChallenge = true
	g.netOnline = true
	g.netEditingName = false
	g.save.Name = "Challenger1"
	g.netFocus = netFocusChallengeStart
	g.render()
	netTxt := screenText(g)
	if !strings.Contains(netTxt, "Options [O]") {
		t.Fatalf("expected Options [O] button in Challenge lobby:\n%s", netTxt)
	}
	if strings.Contains(netTxt, "Change Name") {
		t.Fatalf("did not expect Change Name button in Challenge lobby:\n%s", netTxt)
	}

	g.handleKey(runeKey('o'))
	if g.view != viewSettings || g.setupMode != setupOptions {
		t.Fatalf("expected O to open setupOptions, got view=%d mode=%q", g.view, g.setupMode)
	}
	// Toggle Placement mode (rowQuickPlace) and edit Your name (rowPlayerName).
	g.setIndex = 2 // rowQuickPlace
	g.handleKey(key(tcell.KeyRight))
	if !g.save.Settings.PreviewPlace {
		t.Fatal("expected PreviewPlace to toggle on")
	}
	g.setIndex = 5 // rowPlayerName
	g.focusSettingCaret()
	g.nameDraft = "NewHero"
	g.setIndex = len(gotRows) - 1 // rowDone
	g.handleKey(key(tcell.KeyEnter))
	if g.view != viewNetwork || !g.netChallenge {
		t.Fatalf("expected Done from Options to return to Challenge lobby, got view=%d netChallenge=%v", g.view, g.netChallenge)
	}
	if g.save.Name != "NewHero" {
		t.Fatalf("expected saved name NewHero, got %q", g.save.Name)
	}

	// 3. Preview first (2-press) vs Quick place (1-press) behavior.
	g.openSettings(modeLocal)
	g.startFromSettings()
	g.rollAt = time.Now().Add(-time.Second)

	// With PreviewPlace = true, cursor starts at 1; pressing '3' previews column 2 without placing.
	before := g.ctl.state().Match.Moves
	g.handleKey(runeKey('3'))
	if g.ctl.state().Match.Moves != before {
		t.Fatalf("expected 1st press in PreviewPlace mode not to place die (moves %d -> %d)", before, g.ctl.state().Match.Moves)
	}
	if g.cursor != 2 {
		t.Fatalf("expected cursor to move to 2 on 1st press, got %d", g.cursor)
	}
	// Pressing '3' a second time confirms placement.
	g.handleKey(runeKey('3'))
	if g.ctl.state().Match.Moves != before+1 {
		t.Fatalf("expected 2nd press in PreviewPlace mode to place die (moves %d -> %d)", before, g.ctl.state().Match.Moves)
	}

	// With PreviewPlace = false (1-press Quick Place), pressing '1' places immediately.
	g.save.Settings.PreviewPlace = false
	g.rollAt = time.Now().Add(-time.Second)
	before = g.ctl.state().Match.Moves
	g.handleKey(runeKey('1'))
	if g.ctl.state().Match.Moves != before+1 {
		t.Fatalf("expected Quick Place to place on 1st press (moves %d -> %d)", before, g.ctl.state().Match.Moves)
	}
}
