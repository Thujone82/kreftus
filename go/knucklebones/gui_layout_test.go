package main

import (
	"math"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestViewportRoundTrip(t *testing.T) {
	for _, sz := range [][2]float64{{1100, 760}, {2200, 1520}, {1920, 1080}, {800, 900}} {
		v := fitViewport(sz[0], sz[1])
		r := v.toScreen(menuItemRect(3))
		x, y := v.toBase(r.cx(), r.cy())
		if !menuItemRect(3).has(x, y) {
			t.Errorf("%v: center of menu item 3 maps back to %.1f,%.1f", sz, x, y)
		}
		if r.X < 0 || r.X+r.W > sz[0] || r.Y+r.H > sz[1] {
			t.Errorf("%v: menu item off screen: %+v", sz, r)
		}
	}
	v := fitViewport(1920, 1080)
	if math.Abs(v.offY) > 1e-9 || v.offX <= 0 {
		t.Errorf("wide window should letterbox sideways: %+v", v)
	}
}

func TestColumnHits(t *testing.T) {
	for c := 0; c < gridCols; c++ {
		for _, top := range []bool{true, false} {
			for k := 0; k < gridRows; k++ {
				r := playCell(top, c, k)
				if got := columnAt(r.cx(), r.cy()); got != c {
					t.Errorf("cell top=%v c=%d k=%d hit column %d", top, c, k, got)
				}
			}
		}
	}
	if got := columnAt(guiPanelX+10, guiBotGrid+10); got != -1 {
		t.Errorf("player panel hit column %d", got)
	}
	if got := columnAt(guiGridX+guiDie/2, guiMidY+17); got != 0 {
		t.Errorf("marker row between the grids should still pick column 0, got %d", got)
	}
}

func TestLayoutsDoNotOverlap(t *testing.T) {
	overlap := func(a, b rect) bool {
		return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
	}
	for i := 0; i+1 < menuCount; i++ {
		if overlap(menuItemRect(i), menuItemRect(i+1)) {
			t.Errorf("menu items %d and %d overlap", i, i+1)
		}
	}
	if last := menuItemRect(menuCount - 1); last.Y+last.H > 640 {
		t.Errorf("menu runs into the credits: %+v", last)
	}
	for n := 1; n <= 6; n++ {
		bs := playButtons(n)
		for i := 0; i+1 < n; i++ {
			if overlap(bs[i], bs[i+1]) {
				t.Errorf("%d play buttons: %d and %d overlap", n, i, i+1)
			}
		}
		if bs[0].X < 0 || bs[n-1].X+bs[n-1].W > baseW {
			t.Errorf("%d play buttons leave the window", n)
		}
		if bs[0].Y < guiBotScore+12 {
			t.Errorf("play buttons overlap the bottom column scores")
		}
	}
	card := resultCard(2)
	for _, b := range resultButtons(card, 3) {
		if b.X < card.X || b.X+b.W > card.X+card.W || b.Y+b.H > card.Y+card.H {
			t.Errorf("result button %+v outside card %+v", b, card)
		}
	}
	c := confirmCard(2)
	for _, b := range confirmButtons(c) {
		if b.X < c.X || b.X+b.W > c.X+c.W {
			t.Errorf("confirm button %+v outside card %+v", b, c)
		}
	}
	for i := 0; i < 7; i++ {
		l, r := settingsArrows(settingsRowRect(i))
		if overlap(l, r) || !settingsRowRect(i).has(l.cx(), l.cy()) || !settingsRowRect(i).has(r.cx(), r.cy()) {
			t.Errorf("settings row %d arrows misplaced", i)
		}
	}
	for _, top := range []bool{true, false} {
		for i := 0; i < 2; i++ {
			if d := trayDieRect(top, i); d.X+d.W > guiGridX {
				t.Errorf("tray die runs into the grid: %+v", d)
			}
		}
	}
}

func TestPlayActions(t *testing.T) {
	g := newTestGame(t, 80, 30)
	g.save.Settings.BestOf = 1
	g.openSettings(modeAI)
	g.startFromSettings()
	s := g.ctl.state()
	acts := g.playActions(s)
	labels := ""
	for _, a := range acts {
		labels += a.label + ","
	}
	if labels != "Log,History,Resign,Leave," {
		t.Fatalf("in play: %s", labels)
	}
	g.ctl.resign()
	g.afterChange()
	acts = g.playActions(s)
	if acts[0].label != "View board" || acts[len(acts)-1].label != "Menu" {
		t.Fatalf("after resigning: %+v", acts)
	}
	g.handleKey(acts[0].key)
	if !g.hideResult || g.playActions(s)[0].label != "Show result" {
		t.Fatal("View board button did not hide the result")
	}
	if ev := settingClick(rowStart); ev.Key() != tcell.KeyEnter {
		t.Fatal("clicking Start game should press Enter")
	}
	if settingClick(rowPort) != nil {
		t.Fatal("clicking the port field should only select it")
	}
}
