package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestRotateAccentWraps(t *testing.T) {
	g := &game{accentIndex: 0}
	g.rotateAccent(-1)
	if g.accentIndex != colorWheelSize-1 {
		t.Fatalf("backward wrap: got %d want %d", g.accentIndex, colorWheelSize-1)
	}
	g.rotateAccent(1)
	if g.accentIndex != 0 {
		t.Fatalf("forward wrap: got %d want 0", g.accentIndex)
	}
	g.accentIndex = 15
	g.rotateAccent(1)
	if g.accentIndex != 0 {
		t.Fatalf("15+1: got %d want 0", g.accentIndex)
	}
}

func TestRotateAccentIncorrectJump(t *testing.T) {
	g := &game{accentIndex: 3}
	g.rotateAccent(-8)
	if g.accentIndex != 11 {
		t.Fatalf("-8 from 3: got %d want 11", g.accentIndex)
	}
	g.accentIndex = 0
	g.rotateAccent(-8)
	if g.accentIndex != 8 {
		t.Fatalf("-8 from 0: got %d want 8", g.accentIndex)
	}
}

func TestPencilCursorSkipsYellow(t *testing.T) {
	g := &game{accentIndex: 15, pencil: true} // accent2 = 4 yellow
	c := g.cursorColor()
	if c == colorWheel[3] || c == colorWheel[4] {
		t.Fatal("pencil cursor must skip yellow")
	}
	g.pencil = false
	if g.cursorColor() != colorWheel[4] {
		t.Fatal("pen cursor may use yellow")
	}
	g.accentIndex = 14 // accent2 = 3 gold
	g.pencil = true
	c = g.cursorColor()
	if c == colorWheel[3] || c == colorWheel[4] {
		t.Fatal("pencil cursor must skip gold and yellow")
	}
	if c != colorWheel[5] {
		t.Fatalf("after gold+yellow skip want yellow-green, got %v", c)
	}
}

func TestAccentStepReversesInPencilMode(t *testing.T) {
	g := &game{}
	if g.accentStep() != 1 {
		t.Fatal("pen mode should step +1")
	}
	g.pencil = true
	if g.accentStep() != -1 {
		t.Fatal("pencil mode should step -1")
	}
	g.pencil = false
	g.shiftHold = true
	if g.accentStep() != -1 {
		t.Fatal("held Shift should step -1 like pencil")
	}
	if !g.pencilActive() {
		t.Fatal("held Shift should be pencil-active")
	}
}

func TestAccentOffsets(t *testing.T) {
	g := &game{accentIndex: 0}
	if g.accent2() != colorWheel[5] {
		t.Fatal("secondary should be +5")
	}
	if g.accent3() != colorWheel[11] {
		t.Fatal("tertiary should be -5 (wrap 0→11)")
	}
	g.accentIndex = 14
	if g.accent2() != colorWheel[3] {
		t.Fatal("secondary wrap 14+5")
	}
	if g.accent3() != colorWheel[9] {
		t.Fatal("tertiary 14-5")
	}
}

func TestSudokuBannerLetterOffsets(t *testing.T) {
	g := &game{accentIndex: 0}
	if g.wheelColor(5) != g.accent2() {
		t.Fatal("S should use accent+5")
	}
	for i := 1; i < 6; i++ {
		if g.wheelColor(5+i) != colorWheel[5+i] {
			t.Fatalf("letter %d offset: want wheel %d", i, 5+i)
		}
	}
}

func TestDigitColorsAvoidWhite(t *testing.T) {
	for d := 1; d <= 9; d++ {
		if digitColor[d] == tcell.ColorWhite || digitColor[d] == digitCompleteColor {
			t.Errorf("digit %d must not use white; white is reserved for a completed number", d)
		}
	}
}
