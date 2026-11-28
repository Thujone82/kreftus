package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestScore(t *testing.T) {
	cases := []struct {
		secret, guess string
		exact         int
		partial       int
	}{
		{"RGBC", "RGBC", 4, 0},
		{"RRGG", "GGRR", 0, 4},
		{"RRRR", "RGBY", 1, 0},
		{"RGBY", "YRGB", 0, 4},
		{"RRBB", "RBRB", 2, 2},
	}
	for _, tc := range cases {
		e, p := score([]byte(tc.secret), []byte(tc.guess))
		if e != tc.exact || p != tc.partial {
			t.Errorf("score(%q, %q) = (%d, %d), want (%d, %d)", tc.secret, tc.guess, e, p, tc.exact, tc.partial)
		}
	}
}

func TestParseSetCode(t *testing.T) {
	got, err := parseSetCode("r22m")
	if err != nil {
		t.Fatalf("parseSetCode(\"r22m\") unexpected error: %v", err)
	}
	if !bytes.Equal(got, []byte("RGGM")) {
		t.Errorf("parseSetCode(\"r22m\") = %q, want %q", got, "RGGM")
	}

	badInputs := []string{"", "rgb", "rgbcm", "rgxz", "1274"}
	for _, in := range badInputs {
		if _, err := parseSetCode(in); err == nil {
			t.Errorf("parseSetCode(%q) expected error, got nil", in)
		}
	}
}

func TestKeyToColor(t *testing.T) {
	cases := []struct {
		r    rune
		want byte
		ok   bool
	}{
		{'r', 'R', true},
		{'R', 'R', true},
		{'g', 'G', true},
		{'B', 'B', true},
		{'c', 'C', true},
		{'M', 'M', true},
		{'y', 'Y', true},
		{'1', 'R', true},
		{'2', 'G', true},
		{'3', 'B', true},
		{'4', 'C', true},
		{'5', 'M', true},
		{'6', 'Y', true},
		{'t', 0, false},
		{'s', 0, false},
		{'l', 0, false},
		{'n', 0, false},
		{'q', 0, false},
		{'0', 0, false},
		{'7', 0, false},
	}
	for _, tc := range cases {
		got, ok := keyToColor(tc.r)
		if got != tc.want || ok != tc.ok {
			t.Errorf("keyToColor(%q) = (%q, %v), want (%q, %v)", tc.r, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseKeyTab(t *testing.T) {
	ev := parseKey('\t', bufio.NewReader(strings.NewReader("")))
	if ev.kind != keyLabelMode {
		t.Errorf("parseKey('\\t') = %+v, want kind=%v", ev, keyLabelMode)
	}
}
