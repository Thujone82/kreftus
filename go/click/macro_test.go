package main

import (
	"testing"
	"time"
)

func TestBuildEmpty(t *testing.T) {
	if got := Build(nil); got != nil {
		t.Fatalf("nil: %#v", got)
	}
	if got := Build([]RawEvent{}); got != nil {
		t.Fatalf("empty: %#v", got)
	}
}

func TestBuildKeepsTravelTimeAndDropsPath(t *testing.T) {
	raw := []RawEvent{
		{Tick: 1000, Kind: Move, X: 10, Y: 20},
		{Tick: 1100, Kind: Move, X: 40, Y: 50},
		{Tick: 1500, Kind: LeftDown, X: 80, Y: 90},
		{Tick: 1700, Kind: LeftUp, X: 80, Y: 90},
	}
	got := Build(raw)
	if len(got) != 2 {
		t.Fatalf("len %d: %+v", len(got), got)
	}
	if got[0].Kind != LeftDown || got[0].Offset != 1500*time.Millisecond || got[0].X != 80 || got[0].Y != 90 {
		t.Fatalf("down %+v", got[0])
	}
	if got[1].Kind != LeftUp || got[1].Offset != 1700*time.Millisecond {
		t.Fatalf("up %+v", got[1])
	}
}

func TestBuildDropsTrailingMoves(t *testing.T) {
	raw := []RawEvent{
		{Tick: 0, Kind: LeftUp, X: 1, Y: 1},
		{Tick: 5, Kind: Move, X: 1, Y: 1},
		{Tick: 15, Kind: Move, X: 2, Y: 2},
		{Tick: 25, Kind: Move, X: 9, Y: 9},
	}
	got := Build(raw)
	if len(got) != 1 || got[0].Kind != LeftUp || got[0].Offset != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestLerpCoord(t *testing.T) {
	span := 200 * time.Millisecond
	if got := lerpCoord(0, 100, 0, span); got != 0 {
		t.Fatalf("start %d", got)
	}
	if got := lerpCoord(0, 100, 100*time.Millisecond, span); got != 50 {
		t.Fatalf("mid %d", got)
	}
	if got := lerpCoord(100, 40, 100*time.Millisecond, span); got != 70 {
		t.Fatalf("back %d", got)
	}
	if got := lerpCoord(0, 100, span, span); got != 100 {
		t.Fatalf("end %d", got)
	}
	if got := lerpCoord(3, 9, 0, 0); got != 9 {
		t.Fatalf("zero span %d", got)
	}
}

func TestBuildWheel(t *testing.T) {
	got := Build([]RawEvent{{
		Tick: 5, Kind: Wheel, X: 9, Y: 8, Wheel: -120,
	}})
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Offset != 5*time.Millisecond || got[0].Kind != Wheel || got[0].Wheel != -120 || got[0].X != 9 || got[0].Y != 8 {
		t.Fatalf("%+v", got[0])
	}
}

func TestParseDelay(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"5", 5 * time.Second},
		{"5s", 5 * time.Second},
		{"1200s", 1200 * time.Second},
		{"80m", 80 * time.Minute},
		{"1h20m10s", time.Hour + 20*time.Minute + 10*time.Second},
		{"1H20M10S", time.Hour + 20*time.Minute + 10*time.Second},
		{"1h 20m 10s", time.Hour + 20*time.Minute + 10*time.Second},
		{"0", 0},
		{"0s", 0},
		{"24h", 24 * time.Hour},
		{"1440m", 24 * time.Hour},
		{"86400s", 24 * time.Hour},
	}
	for _, tc := range cases {
		got, err := parseDelay(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDelayRejects(t *testing.T) {
	for _, in := range []string{"", "   ", "1h20", "1x", "abc", "24h1s", "86401", "86401s", "1h20m10"} {
		if _, err := parseDelay(in); err == nil {
			t.Fatalf("%q should fail", in)
		}
	}
}

func TestOverRecordLimit(t *testing.T) {
	if overRecordLimit(maxRecordDuration-time.Millisecond, maxRecordEvents-1) {
		t.Fatal("under both caps")
	}
	if !overRecordLimit(maxRecordDuration, 0) {
		t.Fatal("time cap")
	}
	if !overRecordLimit(0, maxRecordEvents) {
		t.Fatal("event cap")
	}
}
