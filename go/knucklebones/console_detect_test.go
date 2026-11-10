package main

import "testing"

func TestNeedAllocConsole(t *testing.T) {
	cases := []struct {
		name               string
		hasWindow, ssh, pty bool
		want               bool
	}{
		{"ssh no window", false, true, true, false},
		{"ssh with window", true, true, false, false},
		{"local pty no window", false, false, true, false},
		{"double-click tui recovery", false, false, false, true},
		{"shared console", true, false, true, false},
	}
	for _, c := range cases {
		if got := needAllocConsole(c.hasWindow, c.ssh, c.pty); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestNeedShowConsole(t *testing.T) {
	if needShowConsole(true, true) {
		t.Fatal("ssh must not ShowWindow")
	}
	if !needShowConsole(true, false) {
		t.Fatal("local console should ShowWindow")
	}
	if needShowConsole(false, false) {
		t.Fatal("no window to show")
	}
}

func TestNeedWin32ConsoleUI(t *testing.T) {
	if needWin32ConsoleUI(false, true) || needWin32ConsoleUI(false, false) {
		t.Fatal("no window means no Win32 console UI")
	}
	if needWin32ConsoleUI(true, true) {
		t.Fatal("ssh must skip Win32 console UI")
	}
	if !needWin32ConsoleUI(true, false) {
		t.Fatal("local console uses Win32 console UI")
	}
}
