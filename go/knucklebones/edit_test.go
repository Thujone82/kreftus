package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestEditLineCaret(t *testing.T) {
	s, pos := "host", 4
	s, pos, _ = editLine(key(tcell.KeyLeft), s, pos, 16, unicodePrint)
	if s != "host" || pos != 3 {
		t.Fatalf("left: %q pos=%d", s, pos)
	}
	s, pos, _ = editLine(runeKey('X'), s, pos, 16, unicodePrint)
	if s != "hosXt" || pos != 4 {
		t.Fatalf("insert: %q pos=%d", s, pos)
	}
	s, pos, _ = editLine(key(tcell.KeyBackspace2), s, pos, 16, unicodePrint)
	if s != "host" || pos != 3 {
		t.Fatalf("backspace: %q pos=%d", s, pos)
	}
	s, pos, _ = editLine(key(tcell.KeyDelete), s, pos, 16, unicodePrint)
	if s != "hos" || pos != 3 {
		t.Fatalf("delete: %q pos=%d", s, pos)
	}
	s, pos, _ = editLine(key(tcell.KeyHome), s, pos, 16, unicodePrint)
	if pos != 0 {
		t.Fatalf("home: pos=%d", pos)
	}
	s, pos, _ = editLine(key(tcell.KeyEnd), s, pos, 16, unicodePrint)
	if pos != 3 {
		t.Fatalf("end: pos=%d", pos)
	}
	// Typed a/d must insert, not move like WASD navigation.
	s, pos, ok := editLine(runeKey('a'), "xy", 1, 16, unicodePrint)
	if !ok || s != "xay" || pos != 2 {
		t.Fatalf("typed a: %q pos=%d ok=%v", s, pos, ok)
	}
}

func unicodePrint(r rune) bool { return r >= 32 }

func TestNormalizeJoinAddr(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  192.168.1.20  ", "192.168.1.20:8647"},
		{"192.168.1.20:", "192.168.1.20:8647"},
		{"192.168.1.20:8888", "192.168.1.20:8888"},
		{"localhost", "localhost:8647"},
		{"localhost:8647", "localhost:8647"},
	}
	for _, c := range cases {
		if got := normalizeJoinAddr(c.in); got != c.want {
			t.Fatalf("normalizeJoinAddr(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
