package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestSolvedBodyPerfectReplacesErrors(t *testing.T) {
	stats, badge := solvedBody("22:23", 0)
	if badge != "Perfect Finish!" {
		t.Fatalf("badge=%q want Perfect Finish!", badge)
	}
	if len(stats) != 1 || stats[0] != "Time:  22:23" {
		t.Fatalf("stats=%q", stats)
	}
}

func TestSolvedBodyAlignsTimeAndErrors(t *testing.T) {
	stats, badge := solvedBody("3:35", 2)
	if badge != "" {
		t.Fatalf("badge=%q want empty", badge)
	}
	if len(stats) != 2 {
		t.Fatalf("stats=%q", stats)
	}
	if stats[0] != "  Time:  3:35" {
		t.Fatalf("time line=%q", stats[0])
	}
	if stats[1] != "Errors:  2" {
		t.Fatalf("errors line=%q", stats[1])
	}
	colon := -1
	for i, r := range stats[0] {
		if r == ':' {
			colon = i
			break
		}
	}
	errColon := -1
	for i, r := range stats[1] {
		if r == ':' {
			errColon = i
			break
		}
	}
	if colon < 0 || colon != errColon {
		t.Fatalf("colons should align: time=%d errors=%d", colon, errColon)
	}
}

func TestDigitFromRuneShiftSymbols(t *testing.T) {
	d, ok := digitFromRune('%')
	if !ok || d != '5' {
		t.Fatalf("Shift+5 as %%: got %q ok=%v", d, ok)
	}
	if !isShiftedDigit('!') || isShiftedDigit('1') {
		t.Fatal("! is shifted 1; '1' is not a shifted symbol")
	}
	d, ok = digitFromRune('7')
	if !ok || d != '7' {
		t.Fatalf("plain 7: got %q ok=%v", d, ok)
	}
	if _, ok := digitFromRune('0'); ok {
		t.Fatal("0 is clear, not a placeable digit")
	}
}

func TestNumpadShiftDigit(t *testing.T) {
	want := map[tcell.Key]byte{
		tcell.KeyEnd:    '1',
		tcell.KeyDown:   '2',
		tcell.KeyPgDn:   '3',
		tcell.KeyLeft:   '4',
		tcell.KeyClear:  '5',
		tcell.KeyCenter: '5',
		tcell.KeyRight:  '6',
		tcell.KeyHome:   '7',
		tcell.KeyUp:     '8',
		tcell.KeyPgUp:   '9',
	}
	for k, d := range want {
		got, ok := numpadShiftDigit(k)
		if !ok || got != d {
			t.Fatalf("%v: got %q ok=%v want %q", k, got, ok, d)
		}
	}
	if _, ok := numpadShiftDigit(tcell.KeyEnter); ok {
		t.Fatal("Enter is not a numpad digit")
	}
}

func TestShiftMarksKeypadNotDedicatedArrows(t *testing.T) {
	if shiftMarksKey(tcell.KeyUp, false) {
		t.Fatal("dedicated Up with Shift should move, not mark")
	}
	if !shiftMarksKey(tcell.KeyUp, true) {
		t.Fatal("keypad 8 (as Up) with Shift should mark")
	}
	if !shiftMarksKey(tcell.KeyEnd, false) {
		t.Fatal("End / keypad 1 should still mark")
	}
	if shiftMarksKey(tcell.KeyEnter, true) {
		t.Fatal("Enter is not a mark")
	}
}

func TestNoteShiftKeepsKeypadAfterFakeShiftUp(t *testing.T) {
	if shiftHeld() {
		t.Skip("Shift is physically down")
	}
	g := &game{lastShiftHeld: time.Now()}
	e := tcell.NewEventKey(tcell.KeyUp, 0, 0)
	g.noteShift(e)
	if !g.shiftHold {
		t.Fatal("NumLock fake Shift-up + keypad 8 should stay in shift-pencil")
	}
	g = &game{lastShiftHeld: time.Now()}
	e = tcell.NewEventKey(tcell.KeyEnd, 0, 0)
	g.noteShift(e)
	if !g.shiftHold {
		t.Fatal("NumLock fake Shift-up + keypad 1 should stay in shift-pencil")
	}
	g = &game{lastShiftHeld: time.Now().Add(-time.Second)}
	e = tcell.NewEventKey(tcell.KeyUp, 0, 0)
	g.noteShift(e)
	if g.shiftHold {
		t.Fatal("stale Shift should not turn arrows into pencil marks")
	}
}

func TestRefreshShiftHoldIgnoresBriefRelease(t *testing.T) {
	if !shiftPollable() {
		t.Skip("Shift poll is Windows-only")
	}
	if shiftHeld() {
		t.Skip("Shift is physically down")
	}
	g := &game{shiftHold: true, lastShiftHeld: time.Now()}
	if g.refreshShiftHold() {
		t.Fatal("brief GetAsyncKeyState miss should not drop shift-pencil")
	}
	if !g.shiftHold {
		t.Fatal("shiftHold should stick through NumLock fake Shift-up")
	}
	g.lastShiftHeld = time.Now().Add(-time.Second)
	if !g.refreshShiftHold() || g.shiftHold {
		t.Fatal("real Shift release after grace should leave pencil UI")
	}
}
