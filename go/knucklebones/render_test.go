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
		for _, v := range []int{viewMenu, viewName, viewJoin, viewGuide, viewStats} {
			g.view = v
			dump(t, g, "view")
		}
		for _, m := range []string{modeAI, modeLocal, setupHost} {
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
