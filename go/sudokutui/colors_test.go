package main

import "testing"

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
