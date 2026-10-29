package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNextMacroNameLowestFreeAndRollover(t *testing.T) {
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.Local)
	if got := NextMacroName(nil, now); got != "Macro 102926 01" {
		t.Fatalf("empty got %q", got)
	}
	existing := []Macro{
		{Name: "Macro 102926 01"},
		{Name: "Macro 102926 03"},
		{Name: "Macro 102826 02"}, // different day
	}
	if got := NextMacroName(existing, now); got != "Macro 102926 02" {
		t.Fatalf("lowest free got %q, want Macro 102926 02", got)
	}

	// Test rollover past 99 -> 100
	var many []Macro
	for i := 1; i <= 99; i++ {
		many = append(many, Macro{Name: NextMacroName(many, now)})
	}
	if got := NextMacroName(many, now); got != "Macro 102926 100" {
		t.Fatalf("after 99 got %q, want Macro 102926 100", got)
	}
}

func TestBuildStepsClickDoubleClickAndDrag(t *testing.T) {
	raw := []RawEvent{
		{Tick: 100, Kind: LeftDown, X: 200, Y: 300},
		{Tick: 150, Kind: LeftUp, X: 201, Y: 300},
		// Second click within double-click threshold -> coalesced into DoubleClick
		{Tick: 300, Kind: LeftDown, X: 201, Y: 301},
		{Tick: 340, Kind: LeftUp, X: 201, Y: 301},
		// Drag from (400,400) to (550,480)
		{Tick: 900, Kind: RightDown, X: 400, Y: 400},
		{Tick: 1200, Kind: RightUp, X: 550, Y: 480},
	}
	steps := BuildSteps(raw, DefaultBuildOptions(true))
	if len(steps) != 2 {
		t.Fatalf("len=%d: %+v", len(steps), steps)
	}
	if steps[0].Type != StepDoubleClick || steps[0].Button != ButtonLeft || steps[0].DelayMs != 100 {
		t.Fatalf("step 0: %+v", steps[0])
	}
	if steps[1].Type != StepDrag || steps[1].Button != ButtonRight || steps[1].X != 400 || steps[1].X2 != 550 || steps[1].HoldMs != 300 {
		t.Fatalf("step 1: %+v", steps[1])
	}
}

func TestBuildStepsKeyChordsAndTextMerge(t *testing.T) {
	raw := []RawEvent{
		// Ctrl+S chord
		{Tick: 50, Kind: KeyDown, VK: vkLControl},
		{Tick: 80, Kind: KeyDown, VK: 'S'},
		{Tick: 120, Kind: KeyUp, VK: 'S'},
		{Tick: 140, Kind: KeyUp, VK: vkLControl},
		// Typing "Hi" with Shift on H
		{Tick: 300, Kind: KeyDown, VK: vkLShift},
		{Tick: 320, Kind: KeyDown, VK: 'H'},
		{Tick: 350, Kind: KeyUp, VK: 'H'},
		{Tick: 360, Kind: KeyUp, VK: vkLShift},
		{Tick: 420, Kind: KeyDown, VK: 'I'},
		{Tick: 460, Kind: KeyUp, VK: 'I'},
		// Enter key
		{Tick: 700, Kind: KeyDown, VK: vkReturn},
		{Tick: 740, Kind: KeyUp, VK: vkReturn},
	}

	steps := BuildSteps(raw, DefaultBuildOptions(true))
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps (chord, text, enter), got %d: %+v", len(steps), steps)
	}
	if steps[0].Type != StepKey || steps[0].Mods != ModCtrl || steps[0].Key != 'S' || steps[0].KeyName != "Ctrl+S" {
		t.Fatalf("chord step: %+v", steps[0])
	}
	if steps[1].Type != StepText || steps[1].Text != "Hi" {
		t.Fatalf("text step: %+v", steps[1])
	}
	if steps[2].Type != StepKey || steps[2].Key != vkReturn || steps[2].KeyName != "Enter" {
		t.Fatalf("enter step: %+v", steps[2])
	}
}

func TestComputeNextRunAndPreview(t *testing.T) {
	base := time.Date(2026, 10, 29, 9, 0, 0, 0, time.Local)
	s := Schedule{
		ID:         "s1",
		MacroID:    "m1",
		Enabled:    true,
		Start:      base.Add(10 * time.Minute),
		Repeat:     RepeatEveryInterval,
		IntervalMs: int64(15 * time.Minute / time.Millisecond),
		End:        EndAfterRuns,
		EndRuns:    3,
	}

	runs := PreviewNextRuns(s, base, 5)
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs capped by EndAfterRuns, got %d: %v", len(runs), runs)
	}
	if !runs[0].Equal(base.Add(10*time.Minute)) || !runs[1].Equal(base.Add(25*time.Minute)) || !runs[2].Equal(base.Add(40*time.Minute)) {
		t.Fatalf("unexpected run sequence: %v", runs)
	}

	// Active window constraint: 10:00 to 17:00
	s.WindowStart = &TimeOfDay{Hour: 10, Minute: 0}
	s.WindowEnd = &TimeOfDay{Hour: 17, Minute: 0}
	next := ComputeNextRun(s, base)
	if next == nil || next.Hour() != 10 || next.Minute() != 0 {
		t.Fatalf("window start adjustment failed: got %v", next)
	}
}

func TestStoreRoundTripAndCorruptBackup(t *testing.T) {
	dir := t.TempDir()
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := st.UpsertMacro(Macro{
		Steps: []Step{{Type: StepClick, Button: ButtonLeft, X: 10, Y: 20, Glide: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name == "" || m.ID == "" || m.Steps[0].ID == "" {
		t.Fatalf("expected defaults populated: %+v", m)
	}

	// Reload from disk
	st2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	macros, _, _, _ := st2.Snapshot()
	if len(macros) != 1 || macros[0].Name != m.Name {
		t.Fatalf("reloaded macros mismatch: %+v", macros)
	}

	// Write corrupt JSON and verify .bak is created on load
	corruptPath := filepath.Join(dir, "macros.json")
	if err := os.WriteFile(corruptPath, []byte("{not-valid-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(corruptPath + ".bak"); err != nil {
		t.Fatalf("expected .bak file for corrupt JSON: %v", err)
	}
}
