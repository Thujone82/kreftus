package main

import (
	"errors"
	"io"
	"testing"
)

func TestChooseMode(t *testing.T) {
	cases := []struct {
		name                  string
		gui, tui, interactive bool
		want                  uiMode
		err                   error
	}{
		{"terminal", false, false, true, modeTUI, nil},
		{"double-click", false, false, false, modeGUI, nil},
		{"-gui from terminal", true, false, true, modeGUI, nil},
		{"-gui from shortcut", true, false, false, modeGUI, nil},
		{"-tui from terminal", false, true, true, modeTUI, nil},
		{"-tui from shortcut", false, true, false, modeTUI, nil},
		{"both flags", true, true, true, modeTUI, errModeConflict},
	}
	for _, c := range cases {
		got, err := chooseMode(c.gui, c.tui, c.interactive)
		if !errors.Is(err, c.err) || (err == nil && got != c.want) {
			t.Errorf("%s: got %v, %v; want %v, %v", c.name, got, err, c.want, c.err)
		}
	}
}

func TestParseFlags(t *testing.T) {
	f, err := parseFlags([]string{"-gui"}, io.Discard)
	if err != nil || !f.gui || f.tui {
		t.Fatalf("-gui: %+v %v", f, err)
	}
	f, err = parseFlags([]string{"--tui"}, io.Discard)
	if err != nil || !f.tui || f.gui {
		t.Fatalf("--tui: %+v %v", f, err)
	}
	if _, err := parseFlags([]string{"-nope"}, io.Discard); err == nil {
		t.Fatal("unknown flag accepted")
	}
	if _, err := parseFlags([]string{"extra"}, io.Discard); err == nil {
		t.Fatal("stray argument accepted")
	}
}
